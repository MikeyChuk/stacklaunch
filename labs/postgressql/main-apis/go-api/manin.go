package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{
		Name:  "MichaelC",
		Email: "michael.Chidube@example.com",
	},
}

func main() {

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/users", usersHandler)

	log.Println("API listening on port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "stacklaunch-go-api",
	})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listUsersHandler(w)
	case http.MethodPost:
		createUserHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listUsersHandler(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, users)
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

	users = append(users, user)

	writeJSON(w, http.StatusCreated, user)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Unable to encode response: %v", err)
	}
}
