package database

import (
	"context"
	"encoding/json"
)

type Database interface {
	CreateRecord(jobID string, originalObjectKey string, ctx context.Context) error
	RecordExists(jobID string, ctx context.Context) (bool, error)
	UpdateRecordStatus(jobID string, status string, ctx context.Context) error
	GetRecordStatus(jobID string, ctx context.Context) (string, error)
	GetObjectAtKey(jobID string, objectKey string, ctx context.Context) (string, error)
	UpdateFailedStatusAndErrorMessage(jobID string, errorMessage string, ctx context.Context) error
	UpdateRecordAtKey(jobID string, objectKey string, value string, ctx context.Context) error
	UpdateCompletedStatusAndResults(jobID string, data json.RawMessage, objectKey string, ctx context.Context) error
	Ping() error
}
