package database

import "context"

type Database interface {
	CreateRecord(jobID string, originalObjectKey string, ctx context.Context) error
	UpdateRecordStatus(jobID string, status string, ctx context.Context) error
	GetRecordStatus(jobID string, ctx context.Context) (string, error)
	GetObjectAtKey(jobID string, objectKey string, ctx context.Context) (string, error)
	UpdateFailedStatusAndErrorMessage(jobID, errorMessage string, ctx context.Context) error
	Ping() error
}
