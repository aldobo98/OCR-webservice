package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

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
		return "", errors.New("Empty values")
	}
	return final_value, nil
}

type Postgres struct {
	db *sql.DB
}

const host_env = "DB_HOST"
const port_env = "DB_PORT"
const user_env = "DB_USER"
const password_env = "DB_PASSWORD"
const dbName_env = "DB_NAME"
const missing_variables = "No %s argument supplied and %s environment variable is unset, please specify one"

func NewPostgres(logger *slog.Logger, host string, port string, user string, password string, dbname string) *Postgres {
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

	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", database_host, database_port, database_user, database_password, database_name))
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		return nil
	}

	if err := db.Ping(); err != nil {
		logger.Error("Failed to reach database", "error", err)
		if err_ := db.Close(); err_ != nil {
			logger.Error("Failure during closing the database connection", "error", err)
		}
		return nil
	}

	return &Postgres{
		db: db,
	}

}
