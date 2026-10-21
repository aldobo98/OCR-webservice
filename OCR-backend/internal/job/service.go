package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/aldobo98/OCR-backend/dto"
	"github.com/aldobo98/OCR-backend/internal/database"
	"github.com/aldobo98/OCR-backend/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	storage storage.Storage
}

type Database struct {
	database database.Database
}

func NewService(storage storage.Storage) (*Service, error) {
	err := storage.EnsureBucket()
	return &Service{storage: storage}, err
}

func NewDatabase(database database.Database) (*Database, error) {
	err := database.Ping()
	return &Database{database: database}, err
}

func createJob(contentType string, logger *slog.Logger, service *Service, database *Database, ctx context.Context) *dto.CreateJobResponse {
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
	presignedRequest, err := service.storage.Get_Upload_URL(60, objectKey)
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
