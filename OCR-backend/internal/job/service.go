package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/jackc/pgx/v5/pgconn"
)

type Storage struct {
	storage.Storage
}

type Database struct {
	database.Database
}

func NewStorage(storage storage.Storage) (*Storage, error) {
	err := storage.EnsureBucket()
	return &Storage{Storage: storage}, err
}

func NewDatabase(database database.Database) (*Database, error) {
	err := database.Ping()
	return &Database{Database: database}, err
}

func getSupportedStatuses() []string {
	return []string{"created", "uploaded", "queued", "preprocessing", "ocr_processing", "layout_processing", "completed", "failed"}
}

func validStatusChange(current string, goal string) bool {
	//Ha a failed állapotot szeretnénk beállítani, oda minden állapotból mehetünk, kivéve a failed állapotból.
	if goal == "failed" {
		return current != "failed"
	}
	//A failed már ki van értékelve, azzal itt nem kell foglalkozni!
	switch current {
	case "created":
		if goal != "uploaded" {
			return false
		}
	case "uploaded":
		if goal != "queued" {
			return false
		}
	case "queued":
		if goal != "ocr_processing" {
			return false
		}
	case "ocr_processing":
		if !slices.Contains([]string{"layout_processing", "preprocessing", "completed"}, goal) {
			return false
		}
	case "preprocessing":
		//Preprocessing után kutya kötelesek vagyunk újra ocr_processinget csinálni
		if goal != "ocr_processing" {
			return false
		}
	case "layout_processing":
		if goal != "completed" {
			return false
		}
	case "completed":
		return false
	}
	return true
}

func isRetryAbleDBError(err error) bool {
	//Az adatbázis hibakódjait leellenőrizzük
	var pgErr *pgconn.PgError
	//Az nem feltétlen hiba, hogy 0 sort kapunk vissza! Vagy nem feltétlen újrapróbálható.
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001": //serialization failure
			return true
		case "40P01": //deadlock detected
			return true
		}
	}
	//Megszűnt kapcsolatra vonatkozó hiba
	if strings.Contains(err.Error(), "connect: connection refused") {
		return true
	}
	return false
}

func databaseRetry[T any](ctx context.Context, operation func(ctx context.Context) (T, error)) (T, error) {
	const maxAttempts = 5
	var result T
	delay := 100 * time.Millisecond
	ctx_, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			//Itt akkor kapunk errort, ha megszakad a kontextus (pld. bontott kapcsolat), vagy lejárt a timeout.
			return result, err
		}
		//Ha hiba nélkül lefutott, akkor visszatérünk a nil hibával
		result, err := operation(ctx_)
		if err == nil {
			return result, nil
		}
		//Ha van nem újrapróbálható hiba vagy lejárt a próbálkozási ablak
		if !isRetryAbleDBError(err) || attempt == maxAttempts {
			return result, err
		}
		//Egyébként újrapróbálkozunk
		//Kell egy időzítő, ami lejár a delay-t követően
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		case <-timer.C:
			//Ha lejár az időzítő, akkor kilépünk a select-ből, és a delay megnő.
		}
		delay *= 2
	}
	return result, errors.New("unexpected end of database retry loop")
}

func createJob(contentType string, storage *Storage, database *Database, ctx context.Context) (*dto.CreateJobResponse, error) {
	// 1. Egyedi azonosító létrehozása
	jobID := uuid.New()

	// 2. Objektum útvonala
	objectKey := fmt.Sprintf("jobs/%s/input/original.%s", jobID, strings.Split(strings.ReplaceAll(contentType, "}", ""), "/")[1])

	// 3. Adatbázisban elhelyezés
	//TODO: Idempotencia? Ha már létezik a job, de hiba történt, és újrapróbálja a frontend??
	_, err := databaseRetry(ctx, func(_ctx context.Context) (struct{}, error) {
		return struct{}{}, database.CreateRecord(jobID.String(), objectKey, _ctx)
	})
	if err != nil {
		return nil, err
	}
	// 4. presigned upload URL generálása
	presignedRequest, err := storage.Get_Upload_URL(60, objectKey)
	if err != nil {
		return nil, err
	}
	return &dto.CreateJobResponse{
		JobID: jobID.String(),
		Upload: dto.ObjectUploadResponse{
			URL:    presignedRequest.URL,
			Fields: presignedRequest.Values,
		},
	}, nil
}

// updateJobStatus mostmár használható mindenféle status módosítására, át kell adni az adatbázis módosító függvényt.
// A függvénynek mi adjuk meg a kontextust, hogy ne kelljen mindenféle random helyen contextet készíteni.
func updateJobStatus(jobID string, status string, database *Database, ctx context.Context, operation func(_ctx context.Context) error) error {
	//Validálnunk kell hogy valid a státusz
	if slices.Index(getSupportedStatuses(), status) == -1 {
		return errors.New("requested new status is not supported")
	}
	//Le kell kérdezni a jelenlegi státuszt, és ellenőrizni kell, hogy azt beállíthatom-e, amit szeretnék
	currentStatus, err := databaseRetry(ctx, func(_ctx context.Context) (string, error) { return database.GetRecordStatus(jobID, _ctx) })
	if currentStatus == status {
		//Idempotencia!
		return nil
	}
	if err != nil {
		return err
	}
	if !validStatusChange(currentStatus, status) {
		return errors.New("requested new status is not a valid change")
	}
	//Végrehajtjuk a módosítást az adatbázisban
	_, err = databaseRetry(ctx, func(_ctx context.Context) (struct{}, error) { return struct{}{}, operation(ctx) })
	return err
}

func createOcrCloudEvent(objectPath string, jobID string, storage *Storage) (event.Event, error) {

	download_url, err := storage.Get_Download_URL(900, objectPath)
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
		CallBackUrl: fmt.Sprintf("%s/internal/jobs/callback", strings.TrimRight(baseUrl, "/")),
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
