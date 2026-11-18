package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aldobo98/OCR-backend/dto"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
)

func getBackendUrl() string {
	url := os.Getenv("BACKEND_URL")
	if url == "" {
		return ""
	}
	return fmt.Sprintf("%s/internal/jobs/callback", strings.TrimRight(url, "/"))
}

func createOcrCloudEvent(objectPath string, jobID string, storage *Storage) (event.Event, error) {

	download_url, err := storage.Get_Download_URL(900, objectPath)
	if err != nil {
		return event.Event{}, errors.New("failed to create object download pre-signed URL")
	}

	backendUrl := getBackendUrl()
	if backendUrl == "" {
		return event.Event{}, errors.New("failed to read baseUrl from env BACKEND_URL")
	}

	cloudEventData := dto.StartCharacterRecognition{
		JobID: jobID,
		Download: dto.ObjectDownloadResponse{
			URL:    download_url.URL,
			Method: download_url.Method,
			Fields: download_url.SignedHeader,
		},
		CallBackUrl: backendUrl,
	}

	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetSource("/ocr-backend")
	cloudEvent.SetType("hu.ocr-backend.image.process")
	cloudEvent.SetID(uuid.New().String())
	err = cloudEvent.SetData(cloudevents.ApplicationJSON, cloudEventData)
	return cloudEvent, err
}

func createLayoutCloudEvent(ocr_result dto.OCRResponse, jobID string) (event.Event, error) {

	url := getBackendUrl()
	if url == "" {
		return event.Event{}, errors.New("failed to read baseUrl from env BACKEND_URL")
	}

	cloudEventData := dto.StartLayoutAnalysis{
		JobID:       jobID,
		OcrData:     ocr_result,
		CallBackUrl: url,
	}

	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetSource("/ocr-backend")
	cloudEvent.SetType("hu.ocr-backend.layout.process")
	cloudEvent.SetID(uuid.New().String())
	err := cloudEvent.SetData(cloudevents.ApplicationJSON, cloudEventData)
	return cloudEvent, err
}

func sendDoneMessage(jobID string, broker *Broker, logger *slog.Logger) {
	sse_message := dto.ServerSentDone{JobID: jobID, Status: "completed"}
	sse_data, err := json.Marshal(sse_message)
	if err != nil {
		logger.Info("Failed to marshal and publish SSE messsage", "message", sse_message, "error", err)
	} else {
		broker.Publish(jobID, sse_data)
		logger.Info("Successfully published SSE message", "message", sse_message)
	}
}

func sendSseMessage(jobID string, message dto.ServerSentEvent, broker *Broker) {
	sse_data, _ := json.Marshal(message)
	broker.Publish(jobID, sse_data)
}

func startOcrProcessing(jobID string, database *Database, storage *Storage, broker *Broker, logger *slog.Logger) error {
	//Le kell ellenőrizni, hogy az original object tényleg létezik-e
	//Lekérdezzük az object nevét
	objectKey, err := databaseRetry(context.Background(), func(_ctx context.Context) (string, error) {
		return database.GetStringValue(jobID, "original_object_key", _ctx)
	})
	logger.Info("retrieved objectKey", "value", objectKey)
	if err != nil || objectKey == "" {
		logger.Error("Failed to get objectKey", "error", err)
		return fmt.Errorf("failed to retrieve objectKey with error %v", err)
	}
	//Leellenőrizzük, hogy az object az objektumtárolóban van-e
	existence, err := storage.ObjectExists(context.Background(), objectKey)
	if err != nil || !existence {
		logger.Error("Failed to verify existence of object", "error", err)
		return fmt.Errorf("failed to verifyobject existence with error %v", err)
	}
	//Kell egy cloudEvent amit átadhatunk a brokernek
	event, err := createOcrCloudEvent(objectKey, jobID, storage)
	if err != nil {
		logger.Error("Failed to create OCR CloudEvent", "error", err)
		return fmt.Errorf("failed to create OCR CloudEvent with error %v", err)
	}
	logger.Info("Generated CloudEvent", "event", event)
	//Beállítjuk az adatbázisban az új státuszt
	err = updateJobStatus(jobID, "ocr_processing", database, context.Background(), func(_ctx context.Context) error { return database.SetStatus(jobID, "ocr_processing", _ctx) })
	if err != nil {
		logger.Error("Failed to update status", "error", err)
		return fmt.Errorf("failed to update database status with error %v", err)
	}
	logger.Info("Successfully updated database status")
	//Elküldjük a frontendnek, hogy az OCR elkezdődött
	sendSseMessage(jobID, dto.ServerSentEvent{Step: "ocr_processing", Status: "in progress"}, broker)

	err = sendCloudEvent(context.Background(), event, logger)
	if err != nil {
		logger.Error("Failed to send CloudEvent", "error", err)
		err = updateJobStatus(jobID, "failed", database, context.Background(), func(_ctx context.Context) error {
			return database.FailWithMessage(jobID, "Failed to send CloudEvent to EventSink", _ctx)
		})
		if err != nil {
			logger.Error("Failed to set failed status and error message in database", "error", err)
			return fmt.Errorf("failed to set status in database with error %v", err)
		}
		sendSseMessage(jobID, dto.ServerSentEvent{Step: "ocr_processing", Status: "failed"}, broker)
	}
	return nil
}

func startLayoutProcessing(jobID string, database *Database, broker *Broker, logger *slog.Logger) error {
	//leellenőrizzük, hogy az OCR eredménye létezik-e
	ocr_data, err := databaseRetry(context.Background(), func(_ctx context.Context) (json.RawMessage, error) {
		return database.GetJsonValue(jobID, "ocr_results", _ctx)
	})
	if err != nil {
		logger.Error("Failed to retrieve ocr results", "error", err)
		return err
	}
	if len(ocr_data) == 0 {
		logger.Error("No OCR data present, cannot layout analyze")
		return fmt.Errorf("no ocr data available for jobID %v", jobID)
	}
	//Ezek után összerakjuk a requestet
	var ocr_results dto.OCRResponse
	err = json.Unmarshal(ocr_data, &ocr_results)
	if err != nil {
		logger.Error("Failed to unmarshal ocr results", "error", err)
		return err
	}
	event, err := createLayoutCloudEvent(ocr_results, jobID)
	if err != nil {
		logger.Error("Failed to create Layout CloudEvent", "error", err)
		fmt.Errorf("failed to create Layout CloudEvent with error %v", err)
	}
	logger.Info("Generated CloudEvent", "event", event)
	//Beállítjuk az adatbázisban az új státuszt
	err = updateJobStatus(jobID, "layout_processing", database, context.Background(), func(_ctx context.Context) error { return database.SetStatus(jobID, "ocr_processing", _ctx) })
	if err != nil {
		logger.Error("Failed to update status", "error", err)
		return fmt.Errorf("failed to update database status with error %v", err)
	}
	//Elküldjük a frontendnek, hogy az OCR elkezdődött
	sendSseMessage(jobID, dto.ServerSentEvent{Step: "layout_processing", Status: "in progress"}, broker)

	err = sendCloudEvent(context.Background(), event, logger)
	if err != nil {
		logger.Error("Failed to send CloudEvent", "error", err)
		err = updateJobStatus(jobID, "failed", database, context.Background(), func(_ctx context.Context) error {
			return database.FailWithMessage(jobID, "Failed to send CloudEvent to EventSink", _ctx)
		})
		if err != nil {
			logger.Error("Failed to set failed status and error message in database", "error", err)
			return fmt.Errorf("failed to set status in database with error %v", err)
		}
		sendSseMessage(jobID, dto.ServerSentEvent{Step: "layout_processing", Status: "failed"}, broker)
	}
	return nil
}

func calculateNextStep(jobID string, step string, status string, db *Database, broker *Broker, storage *Storage, logger *slog.Logger) {
	//le kell kérni a dokumentum struktúráját, hogy tudjuk, vége van-e

	row, err := databaseRetry(context.Background(), func(_ctx context.Context) (dto.DatabaseRow, error) { return db.GetRow(jobID, _ctx) })
	if err != nil {
		logger.Error("Failed to retrieve document structure from database", "error", err)
	}
	//Kész vagyunk, ha a struktúra unstructured és a step ocr_processing
	if row.Status == "completed" {
		if step == "ocr_processing" {
			if row.DocumentType == "unstructured" {
				//Feltéve, ha van az adatbázisban eredmény
				if len(row.OcrResult) != 0 {
					sendDoneMessage(jobID, broker, logger)
					return
				}
				// TODO: egyébként itt valami baj van.... majd foglalkozom vele ha kell :D
			} else {
				//ha structured, akkor kell a layout analysis
				if len(row.OcrResult) != 0 {
					_ = startLayoutProcessing(jobID, db, broker, logger)
					// TODO: Nemigazán érdekel az error, talán nem kell visszaküldeni? Majd kitalálom később :D
				}
				// TODO: itt valami baj van, de majd foglalkozom vele, ha kell :D
			}
		} else if step == "layout_processing" {
			if len(row.LayoutResult) != 0 {
				sendDoneMessage(jobID, broker, logger)
				return
			}
			// TODO: itt is baj van, de majd foglalkozom vele, ha kell :D
		}
		// TODO: hozzá kell adni a preprocessinget is majd, most nem érdekel
	} else if row.Status == "failed" {
		if step == "ocr_processing" {
			_ = startOcrProcessing(jobID, db, storage, broker, logger)
		} else if step == "layout_processing" {
			_ = startLayoutProcessing(jobID, db, broker, logger)
		}
	}
}
