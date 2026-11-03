package job

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

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
		response, err := createJob(request.ContentType, request.DocType, storage, database, r.Context())

		if err != nil {
			logger.Error("Error during creating job", "error", err)
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

func CreateOcrStartHandler(logger *slog.Logger, storage *Storage, database *Database, broker *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("jobID")
		//2. Meghívjuk a logikát, hogy updateljük a státuszt
		status_to_update := "uploaded"
		err := updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error { return database.SetStatus(jobID, status_to_update, _ctx) })
		//Hogyha valami nem jó, vissza kell küldeni a problémát
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		sse_message := dto.ServerSentEvent{Step: "uploading", Status: "completed"}
		sse_data, err := json.Marshal(sse_message)
		if err != nil {
			logger.Info("Failed to marshal and publish SSE message", "message", sse_message, "error", err)
		} else {
			broker.Publish(jobID, sse_data)
			logger.Info("Published SSE message", "message", sse_message)
		}
		//Le kell ellenőrizni, hogy az original object tényleg létezik-e
		//Lekérdezzük az object nevét
		objectKey, err := databaseRetry(r.Context(), func(_ctx context.Context) (string, error) {
			return database.GetStringValue(jobID, "original_object_key", _ctx)
		})
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
		err = updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error { return database.SetStatus(jobID, "ocr_processing", _ctx) })
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Successfully updated database status")
		//Elküldjük a frontendnek, hogy az OCR elkezdődött
		sse_message = dto.ServerSentEvent{Step: "ocr_processing", Status: "in progress"}
		sse_data, err = json.Marshal(sse_message)
		if err != nil {
			logger.Info("Failed to marshal and publish SSE message", "message", sse_message, "error", err)
		} else {
			broker.Publish(jobID, sse_data)
			logger.Info("Published SSE message", "message", sse_message)
		}
		//Elküldjük az eseményt az eventSinkAddress-re
		err = sendCloudEvent(r.Context(), event, logger)
		if err != nil {
			logger.Error("Failed to send CloudEvent", "error", err)
			status_to_update = "failed"
			err = updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error {
				return database.FailWithMessage(jobID, "Failed to send CloudEvent to EventSink", _ctx)
			})
			if err != nil {
				logger.Error("Failed to set failed status and error message in database", "error", err)
			}
			sse_message = dto.ServerSentEvent{Step: "ocr_processing", Status: "failed"}
			sse_data, err = json.Marshal(sse_message)
			if err != nil {
				logger.Info("Failed to marshal and publish SSE message", "message", sse_message, "error", err)
			} else {
				broker.Publish(jobID, sse_data)
				logger.Info("Published SSE message", "message", sse_message)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Successfully sent event to broker")
		w.WriteHeader(http.StatusAccepted)
	}
}

func CreateCallBackHandler(logger *slog.Logger, storage *Storage, database *Database, broker *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Callback arrived!")
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
		var request dto.CallbackRequest
		err = json.Unmarshal(body, &request)
		if err != nil {
			logger.Error("Failed to unmarshal request", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//Callback request validálása
		callres, err := validate_callback_request(request, database, storage, r.Context())
		if err != nil {
			logger.Error("Callback request was invalid", "error", err, "returncode", convertToReturnCode(callres))
			w.WriteHeader(convertToReturnCode(callres))
			return
		}
		logger.Info("Callback request returncode", "returncode", convertToReturnCode(callres))
		if callres == CallbackIgnored {
			//Idempotencia, nem akarunk ezzel tovább foglalkozni
			logger.Info("Idempotent call is sent back", "code", convertToReturnCode(callres))
			w.WriteHeader(convertToReturnCode(callres))
			return
		}

		logger.Info("Callback request", "request", request)
		jobID := request.JobID

		//Itt már minden valid, csak csináljuk a csinálnivalót
		switch request.Status {
		case "failed":
			//DB update és statuscode visszaküldése
			err = updateJobStatus(jobID, "failed", database, r.Context(), func(_ctx context.Context) error {
				return database.FailWithMessage(jobID, request.ErrorMessage, _ctx)
			})
			if err != nil {
				logger.Error("Failed to update database after knative function reported failure", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "completed":
			//Ha completed, akkor a data-t fel kell tölteni
			var key string
			//Ennek a switchnek a végén kell legyen value
			switch request.Step {
			case "ocr_processing":
				key = "ocr_results"
			case "layout_processing":
				key = "layout_results"
			}
			err = updateJobStatus(jobID, "completed", database, r.Context(), func(_ctx context.Context) error {
				return database.CompleteWithResult(jobID, request.Data, key, _ctx)
			})
			if err != nil {
				logger.Error("Failed to update database with completed status and data", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			logger.Info("Completed callback is sent back with 200OK")
			w.WriteHeader(http.StatusOK)
		}
		sse_message := dto.ServerSentEvent{Step: request.Step, Status: request.Status}
		sse_data, err := json.Marshal(sse_message)
		if err != nil {
			logger.Info("Failed to marshal and publish SSE messsage", "message", sse_message, "error", err)
		} else {
			broker.Publish(jobID, sse_data)
			logger.Info("Successfully published SSE message", "message", sse_message)
		}
	}
}

func CreateEventsHandler(logger *slog.Logger, broker *Broker) http.HandlerFunc {
	//Minden endpoint hívásra saját goroutine indul el, így itt írom meg a végtelen ciklust is
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		jobID := r.PathValue("jobID")
		channel := broker.Subscribe(jobID)
		defer broker.Unsubscribe(jobID, channel)
		for {
			select {
			case <-r.Context().Done():
				return

			case event := <-channel:
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}

				logger.Info("An event was sent to a subscribed frontend", "jobID", jobID, "data", data)
				flusher.Flush()
			}
		}
	}
}
