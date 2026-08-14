package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

// type User struct {
// 	ID    string `json:"id"`
// 	Name  string `json:"name"`
// 	Email string `json:"email"`
// }

type User struct {
	ID      string `json:"id" firestore:"-"`
	Name    string `json:"name" firestore:"name"`
	Email   string `json:"email" firestore:"email"`
	Enabled bool   `json:"enabled" firestore:"enabled"`
}

var firestoreClient *firestore.Client

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/users", usersHandler)
	mux.HandleFunc("/notifications", notificationsHandler)

	// addr := ":8080"

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	addr := ":" + port

	log.Printf("starting StackLaunch API on %s", addr)

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "stacklaunch-firebase-dev")
	if err != nil {
		log.Fatalf("failed to create Firestore client: %v", err)
	}

	defer client.Close()

	firestoreClient = client

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {

	case http.MethodGet:
		ctx := r.Context()

		doc, err := firestoreClient.Collection("users").Doc("user-001").Get(ctx)
		if err != nil {
			http.Error(w, "failed to read user", http.StatusInternalServerError)
			return
		}

		var user User

		if err := doc.DataTo(&user); err != nil {
			http.Error(w, "failed to decode user", http.StatusInternalServerError)
			return
		}

		user.ID = doc.Ref.ID

		json.NewEncoder(w).Encode(user)

	case http.MethodPost:
		var user User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		ref := firestoreClient.Collection("users").NewDoc()

		_, err := ref.Set(ctx, map[string]interface{}{
			"name":    user.Name,
			"email":   user.Email,
			"enabled": user.Enabled,
		})

		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		user.ID = ref.ID

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// func usersHandler(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")

// 	switch r.Method {
// 	case http.MethodGet:
// 		user := User{
// 			ID:    "123",
// 			Name:  "Test User",
// 			Email: "test@example.com",
// 		}

// 		json.NewEncoder(w).Encode(user)

// 	case http.MethodPost:
// 		var user User

// 		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
// 			http.Error(w, "invalid request body", http.StatusBadRequest)
// 			return
// 		}

// 		w.WriteHeader(http.StatusCreated)
// 		json.NewEncoder(w).Encode(user)

// 	default:
// 		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
// 	}
// }

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "notification accepted",
	})
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
