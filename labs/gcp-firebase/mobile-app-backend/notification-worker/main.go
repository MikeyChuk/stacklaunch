package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

var messagingClient *messaging.Client

type PubSubEnvelope struct {
	Message struct {
		Data string `json:"data"`
	} `json:"message"`
}

type NotificationEvent struct {
	FCMToken string `json:"fcm_token"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

func main() {
	ctx := context.Background()

	config := &firebase.Config{
		ProjectID: "stacklaunch-firebase-dev",
	}

	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatalf("firebase init failed: %v", err)
	}

	messagingClient, err = app.Messaging(ctx)
	if err != nil {
		log.Fatalf("messaging client failed: %v", err)
	}

	http.HandleFunc("/", pubSubHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("notification worker listening on :%s", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func pubSubHandler(w http.ResponseWriter, r *http.Request) {
	var envelope PubSubEnvelope

	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, "bad pubsub message", http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		http.Error(w, "bad message data", http.StatusBadRequest)
		return
	}

	var event NotificationEvent

	if err := json.Unmarshal(data, &event); err != nil {
		http.Error(w, "bad notification event", http.StatusBadRequest)
		return
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: event.Title,
			Body:  event.Body,
		},
		Token: event.FCMToken,
	}

	id, err := messagingClient.Send(r.Context(), message)
	if err != nil {
		log.Printf("FCM send failed: %v", err)
		http.Error(w, "FCM failed", http.StatusInternalServerError)
		return
	}

	log.Printf("FCM sent: %s", id)

	w.WriteHeader(http.StatusNoContent)
}
