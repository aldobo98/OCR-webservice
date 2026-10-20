package database

import "context"

type Database interface {
	CreateRecord(jobID string, originalObjectKey string, ctx context.Context) error
	Ping() error
}
