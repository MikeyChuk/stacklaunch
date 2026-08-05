package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *pgxpool.Pool

func main() {
	ctx := context.Background()

	databaseURL := buildDatabaseURL()

	var err error

	db, err = pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Unable to create database connection pool: %v", err)
	}
	defer db.Close()

	if err := waitForDatabase(ctx); err != nil {
		log.Fatalf("Database unavailable: %v", err)
	}

	if err := createUsersTable(ctx); err != nil {
		log.Fatalf("Unable to create users table: %v", err)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/users", usersHandler)

	log.Println("API listening on port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func buildDatabaseURL() string {

	host := getRequiredEnv("DB_HOST")
	port := getEnv("DB_PORT", "5432")
	user := getRequiredEnv("DB_USER")
	password := getRequiredEnv("DB_PASSWORD")
	database := getEnv("DB_NAME", "postgres")
	sslMode := getEnv("DB_SSLMODE", "require")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		host,
		port,
		database,
		sslMode,
	)

	// host := getEnv("DB_HOST", "localhost")
	// port := getEnv("DB_PORT", "5432")
	// user := getEnv("DB_USER", "stacklaunch_admin")
	// password := getEnv("DB_PASSWORD", "stacklaunch_password")
	// database := getEnv("DB_NAME", "stacklaunch")

	// host := getRequiredEnv("DB_HOST")
	// port := getEnv("DB_PORT", "5432")
	// user := getRequiredEnv("DB_USER")
	// password := getRequiredEnv("DB_PASSWORD")
	// database := getRequiredEnv("DB_NAME")

	// return fmt.Sprintf(
	// 	"postgres://%s:%s@%s:%s/%s?sslmode=disable",
	// 	user,
	// 	password,
	// 	host,
	// 	port,
	// 	database,
	// )
}

func waitForDatabase(ctx context.Context) error {
	var lastErr error

	for attempt := 1; attempt <= 20; attempt++ {
		if err := db.Ping(ctx); err == nil {
			log.Println("Connected to PostgreSQL")
			return nil
		} else {
			lastErr = err
		}

		log.Printf("Waiting for PostgreSQL, attempt %d/20", attempt)
		time.Sleep(3 * time.Second)
	}

	return lastErr
}

func createUsersTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(ctx, query)
	return err
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := db.Ping(r.Context()); err != nil {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createUserHandler(w, r)
	case http.MethodGet:
		listUsersHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if user.Name == "" || user.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	var id int
	var createdAt time.Time

	query := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, created_at;
	`

	err := db.QueryRow(
		r.Context(),
		query,
		user.Name,
		user.Email,
	).Scan(&id, &createdAt)

	if err != nil {
		log.Printf("Unable to insert user: %v", err)
		http.Error(w, "Unable to create user", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         id,
		"name":       user.Name,
		"email":      user.Email,
		"created_at": createdAt,
	})
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(
		r.Context(),
		`SELECT id, name, email, created_at FROM users ORDER BY id`,
	)
	if err != nil {
		http.Error(w, "Unable to retrieve users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := make([]map[string]any, 0)

	for rows.Next() {
		var id int
		var name string
		var email string
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &email, &createdAt); err != nil {
			http.Error(w, "Unable to read users", http.StatusInternalServerError)
			return
		}

		users = append(users, map[string]any{
			"id":         id,
			"name":       name,
			"email":      email,
			"created_at": createdAt,
		})
	}

	writeJSON(w, http.StatusOK, users)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Unable to encode response: %v", err)
	}
}

func getRequiredEnv(name string) string {
	value := os.Getenv(name)

	if value == "" {
		log.Fatalf("Required environment variable %s is missing", name)
	}

	return value
}

func getEnv(name string, fallback string) string {
	value := os.Getenv(name)

	if value == "" {
		return fallback
	}

	return value
}
