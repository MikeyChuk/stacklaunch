package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	sslMode := os.Getenv("DB_SSLMODE")

	required := map[string]string{
		"DB_HOST":     host,
		"DB_PORT":     port,
		"DB_NAME":     name,
		"DB_USER":     user,
		"DB_PASSWORD": password,
	}

	for variable, value := range required {
		if value == "" {
			return nil, fmt.Errorf("%s environment variable is missing", variable)
		}
	}

	if sslMode == "" {
		sslMode = "require"
	}

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		name,
		sslMode,
	)

	database, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return database, nil
}
