package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aldobo98/OCR-backend/dto"
	"github.com/aldobo98/ocr-common/aws_s3"
	"github.com/google/uuid"
)

//HTTP handler függvények

func CreateJobRequestHandler(logger *slog.Logger, storage *aws_s3.AWS_S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		//Megkapjuk jsonban a CreateJobRequest DTO-t, amiből filenevet kell generálni
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
		//A jobId és filename generálása után kérni kell a storage-től egy presigned URL-t a feltöltésre
		jobId := uuid.New()
		extension := strings.Split(strings.ReplaceAll(request.ContentType, "}", ""), "/")[1]
		objectKey := fmt.Sprintf("jobs/%s/input/original.%s", jobId, extension)

		presignedPost, err := storage.Get_Upload_URL(60, objectKey)
		if err != nil {
			logger.Error("Failed to get upload URL", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//A presigned URL-t bele kell tenni
		data, err := json.Marshal(dto.CreateJobResponse{
			JobID: jobId.String(),
			Upload: dto.CreateJobUpload{
				URL:    presignedPost.URL,
				Fields: presignedPost.Values,
			},
		})
		if err != nil {
			logger.Error("Failed to marshal response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//Visszaküldjük a választ
		_, err = w.Write(data)
		if err != nil {
			logger.Error("Failed to write response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
