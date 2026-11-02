package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getEnvironmentVariable(name string) string {
	if os.Getenv(name) != "" {
		return os.Getenv(name)
	}
	return ""
}

func choosePriority(p1 string, p2 string) string {
	if p1 != "" {
		return p1
	}
	return p2
}

func getValueBasedOnPriority(argument string, env_variable string) string {
	env_value := getEnvironmentVariable(env_variable)
	return choosePriority(env_value, argument)
}

type Postgres struct {
	db *sql.DB
}

const host_env = "DB_HOST"
const port_env = "DB_PORT"
const user_env = "DB_USER"
const password_env = "DB_PASSWORD"
const dbName_env = "DB_NAME"

func NewPostgres(host string, port string, user string, password string, dbname string, table string) (*Postgres, error) {
	database_host := getValueBasedOnPriority(host, host_env)
	if database_host == "" {
		return nil, errors.New("no database host specified")
	}
	database_port := getValueBasedOnPriority(port, port_env)
	if database_port == "" {
		return nil, errors.New("no database port specified")
	}
	database_user := getValueBasedOnPriority(user, user_env)
	if database_user == "" {
		return nil, errors.New("no database user specified")
	}
	database_password := getValueBasedOnPriority(password, password_env)
	if database_password == "" {
		return nil, errors.New("no database password specified")
	}
	database_name := getValueBasedOnPriority(dbname, dbName_env)
	if database_name == "" {
		return nil, errors.New("no database name is specified")
	}

	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", database_host, database_port, database_user, database_password, database_name))
	if err != nil {
		return nil, err
	}

	return &Postgres{
		db: db,
	}, nil
}

func (p *Postgres) Ping() error {
	return p.db.Ping()
}

func (p *Postgres) CreateRecord(jobID string, originalObjectKey string, fileExtension string, documentType string, ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `INSTERT INTO jobs 
									(id, original_object_key, original_file_extension, document_type)
									VALUES ($1, $2, $3, $4)`,
		jobID, originalObjectKey, fileExtension, documentType)
	return err
}

func (p *Postgres) SetStatus(jobID string, status string, ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `UPDATE jobs
									SET status = $2
									WHERE id = $1`,
		jobID, status)
	return err
}

func (p *Postgres) GetStringValue(jobID string, key string, ctx context.Context) (string, error) {
	var object sql.NullString
	err := p.db.QueryRowContext(ctx, fmt.Sprintf("SELECT %s FROM jobs WHERE id = $1", key), jobID).Scan(&object)
	if !object.Valid {
		return "", nil
	}
	return object.String, err
}

func (p *Postgres) GetStatus(jobID string, ctx context.Context) (string, error) {
	var status string
	err := p.db.QueryRowContext(ctx, `SELECT status FROM jobs WHERE ID = $1`, jobID).Scan(&status)
	return status, err
}

func (p *Postgres) FailWithMessage(jobID string, errorMessage string, ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `UPDATE jobs
									SET status = 'failed', error_message = $2
									WHERE id = $1`, jobID, errorMessage)
	return err
}

func (p *Postgres) SetStringValue(jobID string, key string, value string, ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("UPDATE jobs SET %s = $2 WHERE id = $1", key), jobID, value)
	return err
}

func (p *Postgres) CompleteWithResult(jobID string, data json.RawMessage, key string, ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("UPDATE jobs SET status = 'completed', %s = $2 WHERE id = $1", key), jobID, data)
	return err
}

func (p *Postgres) RecordExists(jobID string, ctx context.Context) (bool, error) {
	var exists bool

	err := p.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM jobs WHERE id = $1)`,
		jobID,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p *Postgres) GetFileExtension(jobID string, ctx context.Context) (string, error) {
	var extension string
	err := p.db.QueryRowContext(ctx, `SELECT original_file_extension FROM jobs WHERE ID = $1`, jobID).Scan(&extension)
	return extension, err
}

func (p *Postgres) GetPreprocessingAttempts(jobID string, ctx context.Context) (int, error) {
	var attempts int
	err := p.db.QueryRowContext(ctx, `SELECT preprocessing_attempts FROM jobs WHERE ID = $1`, jobID).Scan(&attempts)
	return attempts, err
}
