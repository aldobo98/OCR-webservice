package job

import (
	"context"
	"fmt"
	"log/slog"
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

//Storage meghívása

//Adatbázis műveletek

//Üzleti szabályok
