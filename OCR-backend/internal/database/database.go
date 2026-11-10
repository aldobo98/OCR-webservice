package database

import (
	"context"
	"encoding/json"

	"github.com/aldobo98/OCR-backend/dto"
)

type Database interface {
	CreateRecord(jobID string, originalObjectKey string, fileExtension string, documentType string, ctx context.Context) error
	RecordExists(jobID string, ctx context.Context) (bool, error)
	SetStatus(jobID string, status string, ctx context.Context) error
	GetStatus(jobID string, ctx context.Context) (string, error)
	GetStringValue(jobID string, key string, ctx context.Context) (string, error)
	FailWithMessage(jobID string, errorMessage string, ctx context.Context) error
	SetStringValue(jobID string, key string, value string, ctx context.Context) error
	CompleteWithResult(jobID string, data json.RawMessage, key string, ctx context.Context) error
	GetFileExtension(jobID string, ctx context.Context) (string, error)
	GetPreprocessingAttempts(jobID string, ctx context.Context) (int, error)
	GetJsonValue(jobID string, key string, ctx context.Context) (json.RawMessage, error)
	GetRow(jobID string, ctx context.Context) (dto.DatabaseRow, error)
	Ping() error
}
