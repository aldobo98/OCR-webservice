package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aldobo98/OCR-backend/dto"
	"github.com/aldobo98/OCR-backend/internal/database"
	"github.com/aldobo98/OCR-backend/internal/storage"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
)

type Storage struct {
	storage storage.Storage
}

type Database struct {
	database database.Database
}

func NewStorage(storage storage.Storage) (*Storage, error) {
	err := storage.EnsureBucket()
	return &Storage{storage: storage}, err
}

func NewDatabase(database database.Database) (*Database, error) {
	err := database.Ping()
	return &Database{database: database}, err
}

func createJob(contentType string, logger *slog.Logger, storage *Storage, database *Database, ctx context.Context) *dto.CreateJobResponse {
	// 1. Egyedi azonosító létrehozása
	jobID := uuid.New()

	// 2. Objektum útvonala
	objectKey := fmt.Sprintf("jobs/%s/input/original.%s", jobID, strings.Split(strings.ReplaceAll(contentType, "}", ""), "/")[1])

	// 3. Adatbázisban elhelyezés
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := database.database.CreateRecord(jobID.String(), objectKey, ctx)
	if err != nil {
		logger.Error("Failed to create database entry", "error", err)
		return nil
	}
	// 4. presigned upload URL generálása
	presignedRequest, err := storage.storage.Get_Upload_URL(60, objectKey)
	if err != nil {
		logger.Error("Failed to create presigned upload URL", "error", err)
		return nil
	}
	return &dto.CreateJobResponse{
		JobID: jobID.String(),
		Upload: dto.ObjectUploadResponse{
			URL:    presignedRequest.URL,
			Fields: presignedRequest.Values,
		},
	}
}

func getSupportedStatuses() []string {
	return []string{"created", "uploaded", "queued", "preprocessing", "ocr_processing", "layout_processing", "completed", "failed"}
}

func updateJobStatus(jobID string, status string, logger *slog.Logger, database *Database, ctx context.Context) (int, error) {
	//Validálnunk kell hogy valid a státusz
	if slices.Index(getSupportedStatuses(), status) == -1 {
		return http.StatusBadRequest, errors.New("requested new status is not supported")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	//Végrehajtjuk a módosítást az adatbázisban
	err := database.database.UpdateRecordStatus(jobID, status, ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func getObjectKey(jobID string, objectKey string, database *Database, ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	//Lekérdezzük az objectet ami az objectKey alatt van
	return database.database.GetObjectAtKey(jobID, objectKey, ctx)
}

func createOcrCloudEvent(objectPath string, jobID string, storage *Storage, logger *slog.Logger) (event.Event, error) {

	download_url, err := storage.storage.Get_Download_URL(120, objectPath)
	if err != nil {
		return event.Event{}, errors.New("failed to create object download pre-signed URL")
	}

	baseUrl := os.Getenv("BACKEND_URL")
	if baseUrl == "" {
		return event.Event{}, errors.New("failed to read baseUrl from env BACKEND_URL")
	}

	cloudEventData := dto.StartCharacterRecognition{
		JobID: jobID,
		Download: dto.ObjectDownloadResponse{
			URL:    download_url.URL,
			Method: download_url.Method,
			Fields: download_url.SignedHeader,
		},
		CallBackUrl: fmt.Sprintf("%s/api/jobs/%s/ocr", strings.TrimRight(baseUrl, "/"), jobID),
	}

	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetSource("/ocr-backend")
	cloudEvent.SetType("hu.ocr-backend.image.process")
	cloudEvent.SetID(uuid.New().String())
	err = cloudEvent.SetData(cloudevents.ApplicationJSON, cloudEventData)
	return cloudEvent, err
}

func sendCloudEvent(ctx context.Context, cloudEvent event.Event, logger *slog.Logger) error {
	//SinkBinding automatikus injektálja a broker címét a K_SINK környezeti változóba
	eventSinkAddress := os.Getenv("K_SINK")
	if eventSinkAddress == "" {
		return errors.New("failed to resolve eventSink address from K_SINK")
	}
	logger.Info("EventSink Address", "address", eventSinkAddress)
	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		return err
	}
	ctx = cloudevents.ContextWithTarget(ctx, eventSinkAddress)
	if result := client.Send(ctx, cloudEvent); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send CloudEvent with error %w", result)
	}
	return nil
}
