package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func getEnvironmentVariable(name string, logger *slog.Logger) string {
	if os.Getenv(name) != "" {
		logger.Info(fmt.Sprintf("Environment variable %s is set, using it's value", name), "value", os.Getenv(name))
		return os.Getenv(name)
	}
	logger.Info(fmt.Sprintf("Environment variable %s is unset", name))
	return ""
}

func choosePriority(p1 string, p2 string) string {
	if p1 != "" {
		return p1
	}
	return p2
}

func getValueBasedOnPriority(argument string, env_variable string, logger *slog.Logger, error_message string) (string, error) {
	env_value := getEnvironmentVariable(env_variable, logger)
	final_value := choosePriority(env_value, argument)
	if final_value == "" {
		logger.Error(error_message)
		return "", errors.New("empty values")
	}
	return final_value, nil
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
const missing_variables = "No %s argument supplied and %s environment variable is unset, please specify one"

func NewPostgres(logger *slog.Logger, host string, port string, user string, password string, dbname string, table string) *Postgres {
	database_host, err := getValueBasedOnPriority(host, host_env, logger, fmt.Sprintf(missing_variables, "host", host_env))
	if err != nil {
		return nil
	}
	database_port, err := getValueBasedOnPriority(port, port_env, logger, fmt.Sprintf(missing_variables, "port", port_env))
	if err != nil {
		return nil
	}
	database_user, err := getValueBasedOnPriority(user, user_env, logger, fmt.Sprintf(missing_variables, "user", user_env))
	if err != nil {
		return nil
	}
	database_password, err := getValueBasedOnPriority(password, password_env, logger, fmt.Sprintf(missing_variables, "password", password_env))
	if err != nil {
		return nil
	}
	database_name, err := getValueBasedOnPriority(dbname, dbName_env, logger, fmt.Sprintf(missing_variables, "dbname", dbName_env))
	if err != nil {
		return nil
	}
	database_table, err := getValueBasedOnPriority(table, table_env, logger, fmt.Sprintf(missing_variables, "table", table_env))
	if err != nil {
		return nil
	}

	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", database_host, database_port, database_user, database_password, database_name))
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		return nil
	}

	db_sanitized_table := pgx.Identifier{database_table}.Sanitize()

	return &Postgres{
		db:    db,
		table: db_sanitized_table,
	}
}

func (p *Postgres) Ping() error {
	return p.db.Ping()
}

const createRecordQuery = "INSERT INTO %s (id, status, original_object_key) VALUES $1 $2 $3"

func (p *Postgres) CreateRecord(jobID string, originalObjectKey string, ctx context.Context) error {
	query := fmt.Sprintf(createRecordQuery, p.table)
	_, err := p.db.ExecContext(ctx, query, jobID, "created", originalObjectKey)
	return err
}
