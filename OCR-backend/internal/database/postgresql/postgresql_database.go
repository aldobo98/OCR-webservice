package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
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
	db    *sql.DB
	table string
}

const host_env = "DB_HOST"
const port_env = "DB_PORT"
const user_env = "DB_USER"
const password_env = "DB_PASSWORD"
const dbName_env = "DB_NAME"
const table_env = "DB_TABLE"

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
	database_table := getValueBasedOnPriority(table, table_env)
	if database_table == "" {
		return nil, errors.New("no database table is specified")
	}

	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", database_host, database_port, database_user, database_password, database_name))
	if err != nil {
		return nil, err
	}

	db_sanitized_table := pgx.Identifier{database_table}.Sanitize()

	return &Postgres{
		db:    db,
		table: db_sanitized_table,
	}, nil
}

func (p *Postgres) Ping() error {
	return p.db.Ping()
}

const createRecordQuery = "INSERT INTO %s (id, status, original_object_key) VALUES ($1, $2, $3)"

func (p *Postgres) CreateRecord(jobID string, originalObjectKey string, ctx context.Context) error {
	query := fmt.Sprintf(createRecordQuery, p.table)
	_, err := p.db.ExecContext(ctx, query, jobID, "created", originalObjectKey)
	return err
}

const updateRecordStatusQuery = "UPDATE %s SET status = $1 WHERE id = $2"

func (p *Postgres) UpdateRecordStatus(jobID string, status string, ctx context.Context) error {
	query := fmt.Sprintf(updateRecordStatusQuery, p.table)
	_, err := p.db.ExecContext(ctx, query, status, jobID)
	return err
}

const getObjectAtKeyQuery = "SELECT %s FROM %s WHERE id = $1"

func (p *Postgres) GetObjectAtKey(jobID string, objectKey string, ctx context.Context) (string, error) {
	query := fmt.Sprintf(getObjectAtKeyQuery, objectKey, p.table)
	var object sql.NullString
	err := p.db.QueryRowContext(ctx, query, jobID).Scan(&object)
	if !object.Valid {
		return "", nil
	}
	return object.String, err
}

const getRecordStatusQuery = "SELECT status FROM %s WHERE id = $1"

func (p *Postgres) GetRecordStatus(jobID string, ctx context.Context) (string, error) {
	query := fmt.Sprintf(getRecordStatusQuery, p.table)
	var status string
	err := p.db.QueryRowContext(ctx, query, jobID).Scan(&status)
	return status, err
}

const setFailedAndMessage = "UPDATE %s SET status = 'failed' , error_message = $1 WHERE id = $2"

func (p *Postgres) UpdateFailedStatusAndErrorMessage(jobID string, errorMessage string, ctx context.Context) error {
	query := fmt.Sprintf(setFailedAndMessage, p.table)
	_, err := p.db.ExecContext(ctx, query, errorMessage, jobID)
	return err
}

const updateRecordAtKeyQuery = "UPDATE %s SET %s = $1 WHERE id = $2"

func (p *Postgres) UpdateRecordAtKey(jobID string, objectKey string, value string, ctx context.Context) error {
	query := fmt.Sprintf(updateRecordAtKeyQuery, p.table, objectKey)
	_, err := p.db.ExecContext(ctx, query, value, jobID)
	return err
}

const updateCompletedAndResult = "UPDATE %s SET status = 'completed, %s = $1 WHERE id = $2"

func (p *Postgres) UpdateCompletedStatusAndResults(jobID string, data json.RawMessage, objectKey string, ctx context.Context) error {
	query := fmt.Sprintf(updateCompletedAndResult, p.table, objectKey)
	_, err := p.db.ExecContext(ctx, query, data, jobID)
	return err
}
