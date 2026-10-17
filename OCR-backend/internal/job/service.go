package job

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/aldobo98/OCR-backend/dto"
	"github.com/aldobo98/OCR-backend/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) (*Service, error) {
	err := storage.EnsureBucket()
	return &Service{storage: storage}, err
}

func createJob(contentType string, logger *slog.Logger, service *Service) *dto.CreateJobResponse {
	// 1. Egyedi azonosító létrehozása
	jobID := uuid.New()

	// 2. Objektum útvonala
	objectKey := fmt.Sprintf("jobs/%s/input/original.%s", jobID, strings.Split(strings.ReplaceAll(contentType, "}", ""), "/")[1])

	// 3. TODO: Adatbázisban elhelyezés

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

//Storage meghívása

//Adatbázis műveletek

//Üzleti szabályok
