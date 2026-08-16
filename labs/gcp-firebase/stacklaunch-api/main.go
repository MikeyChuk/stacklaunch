package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
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

// Add a global auth client:
var authClient *auth.Client
var firestoreClient *firestore.Client

// Add a global storage client: storage SDK is not used in this example, but you can use it to interact with Google Cloud Storage if needed.
var storageClient *storage.Client

type contextKey string

const uidKey contextKey = "uid"

// Add a constant for the bucket name:
const bucketName = "stacklaunch-firebase-dev.firebasestorage.app"

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/users", requireAuth(usersHandler))
	mux.HandleFunc("/notifications", notificationsHandler)
	// Add the /files endpoint with authentication
	mux.HandleFunc("/files", requireAuth(filesHandler))

	// addr := ":8080"

	port := getEnv("PORT", "8080")

	addr := ":" + port

	log.Printf("starting StackLaunch API on %s", addr)

	ctx := context.Background()

	// Firebase app
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("failed to initialize Firebase app: %v", err)
	}

	// Firebase Authentication client
	authClientLocal, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("failed to initialize Firebase Auth client: %v", err)
	}

	authClient = authClientLocal

	// Firestore client
	firestoreClientLocal, err := firestore.NewClient(
		ctx,
		"stacklaunch-firebase-dev",
	)
	if err != nil {
		log.Fatalf("failed to create Firestore client: %v", err)
	}

	defer firestoreClientLocal.Close()

	firestoreClient = firestoreClientLocal

	// Storage client
	storageClientLocal, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create Storage client: %v", err)
	}

	defer storageClientLocal.Close()

	storageClient = storageClientLocal

	// Start API
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
	uid, ok := r.Context().Value(uidKey).(string)
	if !ok || uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {

	case http.MethodGet:
		ctx := r.Context()

		// doc, err := firestoreClient.Collection("users").Doc("user-001").Get(ctx)

		doc, err := firestoreClient.Collection("users").Doc(uid).Get(ctx)
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

		ref := firestoreClient.Collection("users").Doc(uid)

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

// auth middleware
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		idToken := parts[1]

		token, err := authClient.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		log.Printf("authenticated Firebase user: %s", token.UID)
		ctx := context.WithValue(r.Context(), uidKey, token.UID)

		next(w, r.WithContext(ctx))

	}
}

// admin middleware

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)

		token, err := authClient.VerifyIDToken(r.Context(), parts[1])
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		admin, ok := token.Claims["admin"].(bool)
		if !ok || !admin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	})
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uid, ok := r.Context().Value(uidKey).(string)
	if !ok || uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	objectName := "users/" + uid + "/" + header.Filename

	wc := storageClient.
		Bucket(bucketName).
		Object(objectName).
		NewWriter(r.Context())

	if _, err := io.Copy(wc, file); err != nil {
		log.Printf("storage upload failed: %v", err)
		http.Error(w, "failed to upload file", http.StatusInternalServerError)
		return
	}

	if err := wc.Close(); err != nil {
		log.Printf("storage finalize failed: %v", err)
		http.Error(w, "failed to finalize upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"object": objectName,
	})
}
