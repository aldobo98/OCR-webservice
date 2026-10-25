package job

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/aldobo98/OCR-backend/dto"
)

//HTTP handler függvények

func CreateJobHandler(logger *slog.Logger, storage *Storage, database *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Kiolvassuk a request tartalmát
		defer func() {
			err := r.Body.Close()
			if err != nil {
				logger.Error("Failed to close message body", "error", err)
			}
		}()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read response body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var request dto.CreateJobRequest
		err = json.Unmarshal(body, &request)
		if err != nil {
			logger.Error("Failed to unmarshal request", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Message body read", "value", request)

		// 2. Meghívjuk a logikát
		response := createJob(request.ContentType, logger, storage, database, r.Context())

		if response == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 3. Válaszolunk
		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			logger.Error("Failed to write response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func CreateOcrStartHandler(logger *slog.Logger, storage *Storage, database *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("jobID")
		//2. Meghívjuk a logikát, hogy updateljük a státuszt
		responseCode, err := updateJobStatus(jobID, "uploaded", database, r.Context())
		//Hogyha valami nem jó, vissza kell küldeni a problémát
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(responseCode)
			return
		}
		//Le kell ellenőrizni, hogy az original object tényleg létezik-e
		//Lekérdezzük az object nevét
		objectKey, err := getObjectKey(jobID, "original_object_key", database, r.Context())
		logger.Info("retrieved objectKey", "value", objectKey)
		if err != nil || objectKey == "" {
			logger.Error("Failed to get objectKey", "error", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		//Leellenőrizzük, hogy az object az objektumtárolóban van-e
		existence, err := storage.ObjectExists(r.Context(), objectKey)
		if err != nil || !existence {
			logger.Error("Failed to verify existence of object", "error", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		//Kell egy cloudEvent amit átadhatunk a brokernek
		event, err := createOcrCloudEvent(objectKey, jobID, storage)
		if err != nil {
			logger.Error("Failed to create OCR CloudEvent", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Generated CloudEvent", "event", event)
		//Beállítjuk az adatbázisban az új státuszt
		responseCode, err = updateJobStatus(jobID, "queued", database, r.Context())
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(responseCode)
			return
		}
		//Elküldjük az eseményt az eventSinkAddress-re
		err = sendCloudEvent(r.Context(), event, logger)
		if err != nil {
			logger.Error("Failed to send CloudEvent", "error", err)
			ctx_, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			err = databaseModifWithRetry(r.Context(), func() error {
				return database.UpdateFailedStatusAndErrorMessage(jobID, "Failed to send CloudEvent to EventSink", ctx_)
			})
			if err != nil {
				logger.Error("Failed to set failed status and error message in database", "error", err)
			}
			w.WriteHeader(http.StatusInternalServerError)
		}
		logger.Info("Successfully sent event to broker")

		logger.Info("Successfully updated database status")
		w.WriteHeader(http.StatusAccepted)
	}
}
