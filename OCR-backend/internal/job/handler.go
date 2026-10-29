package job

import (
	"context"
	"encoding/json"
	"fmt"
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
		response, err := createJob(request.ContentType, storage, database, r.Context())

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

func CreateOcrStartHandler(logger *slog.Logger, storage *Storage, database *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("jobID")
		//2. Meghívjuk a logikát, hogy updateljük a státuszt
		status_to_update := "uploaded"
		err := updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error { return database.UpdateRecordStatus(jobID, status_to_update, _ctx) })
		//Hogyha valami nem jó, vissza kell küldeni a problémát
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
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
		status_to_update = "queued"
		err = updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error { return database.UpdateRecordStatus(jobID, status_to_update, _ctx) })
		if err != nil {
			logger.Error("Failed to update status", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("Successfully updated database status")
		//Elküldjük az eseményt az eventSinkAddress-re
		err = sendCloudEvent(r.Context(), event, logger)
		if err != nil {
			logger.Error("Failed to send CloudEvent", "error", err)
			status_to_update = "failed"
			err = updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error {
				return database.UpdateFailedStatusAndErrorMessage(jobID, "Failed to send CloudEvent to EventSink", _ctx)
			})
			if err != nil {
				logger.Error("Failed to set failed status and error message in database", "error", err)
			}
			w.WriteHeader(http.StatusInternalServerError)
		}
		logger.Info("Successfully sent event to broker")
		w.WriteHeader(http.StatusAccepted)
	}
}

func CreateCallBackHandler(logger *slog.Logger, storage *Storage, database *Database) http.HandlerFunc {
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

		var response dto.CallbackResponse

		//Callback request validálása
		err = validate_callback_request(request, database, storage, r.Context())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("Callback request was invalid", "error", err)
			response.Message = fmt.Sprintf("Callback request validation failed with the following error: %s", err)
			write_callback_response(response, &w, logger)
			return
		}

		logger.Info("Callback request data", "data", response)
		jobID := request.JobID

		//Itt már minden valid, csak csináljuk a csinálnivalót
		switch request.Status {
		case "failed":
			//DB update és statuscode visszaküldése
			status_to_update := "failed"
			err = updateJobStatus(jobID, status_to_update, database, r.Context(), func(_ctx context.Context) error {
				return database.UpdateFailedStatusAndErrorMessage(jobID, request.ErrorMessage, _ctx)
			})
			if err != nil {
				logger.Error("Failed to update database after knative function reported failure", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				response.Message = "Failed to update database with status and error message"
				write_callback_response(response, &w, logger)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		case "starting":
			//Ha vannak requestek, akkor először le kell generálni a presigned URL-eket
			objectRequest := request.Requests
			if objectRequest != (dto.ObjectRequest{}) {
				//Presigned GET, download request
				re := objectRequest.Read
				if re != "" {
					//Ki kell nyerni, amire szeretnénk download linket: original vagy preprocess
					//Nem kell ellenőrizgetni semmit, már valid a request
					path, err := queryJobValue(r.Context(), func(_ctx context.Context) (string, error) {
						return database.GetObjectAtKey(jobID, fmt.Sprintf("%s_object_key", re), _ctx)
					})
					if err != nil {
						logger.Error("Failed to read path from database", "error", err)
						w.WriteHeader(http.StatusInternalServerError)
						response.Message = "Failed to read the path of the requested read object"
						write_callback_response(response, &w, logger)
						return
					}
					download_url, err := storage.Get_Download_URL(1800, path)
					if err != nil {
						logger.Error("Failed to get download URL", "error", err)
						w.WriteHeader(http.StatusInternalServerError)
						response.Message = "Failed to generate download URL for requested read object"
						write_callback_response(response, &w, logger)
					}
					response.Read = dto.ObjectDownloadResponse{URL: download_url.URL, Method: download_url.Method, Fields: download_url.SignedHeader}
				}
				//Presigned POST, upload request
				wr := objectRequest.Write
				if wr != "" {
					//A request valid volt, nem kell adatbázist lekérdezni, és a path-ot le kell generálni
					path := fmt.Sprintf("jobs/%s/input/preprocessed", jobID)
					//Be kell tenni az adatbázisba
					err := updateJobValue(r.Context(), func(_ctx context.Context) error {
						return database.UpdateRecordAtKey(jobID, "preprocess_object_key", path, _ctx)
					})
					if err != nil {
						logger.Error("Failed to update database at preprocess_object_key", "error", err)
						response.Message = "Failed to update database at preprocess_object_key"
						write_callback_response(response, &w, logger)
						return
					}
					//Kérünk presigned URL-t
					upload_url, err := storage.Get_Upload_URL(1800, path)
					if err != nil {
						logger.Error("Failed to get presigned URL", "error", err)
						response.Message = "Failed to get presigned URL for requested write object"
						write_callback_response(response, &w, logger)
						return
					}
					response.Write = dto.ObjectUploadResponse{URL: upload_url.URL, Fields: upload_url.Values}
				}
			}
			//Fel kell venni az adatbázisba a státuszváltozást
			//Nem a "starting" statuszt akarom felvenni, hanem hogy az adott lépésben vagyok!
			err := updateJobStatus(jobID, request.Step, database, r.Context(), func(_ctx context.Context) error { return database.UpdateRecordStatus(jobID, request.Step, _ctx) })
			if err != nil {
				logger.Error("Failed to update database", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				response.Message = "Failed to update database with new status"
				write_callback_response(response, &w, logger)
				return
			}
			w.WriteHeader(http.StatusOK)
			//És akkor most tudunk válaszolni
			write_callback_response(response, &w, logger)
			return
		case "completed":
			//Ha completed, akkor a data-t fel kell tölteni
			var objectKey string
			//Ennek a switchnek a végén kell legyen value
			switch request.Step {
			case "ocr_processing":
				objectKey = "ocr_results"
			case "layout_processing":
				objectKey = "layout_results"
			}
			err = updateJobStatus(jobID, "completed", database, r.Context(), func(_ctx context.Context) error {
				return database.UpdateCompletedStatusAndResults(jobID, request.Data, objectKey, _ctx)
			})
			if err != nil {
				logger.Error("Failed to update database with completed status and data", "error", err)
				response.Message = "Failed to update database with completed status and data"
				w.WriteHeader(http.StatusInternalServerError)
				write_callback_response(response, &w, logger)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}
