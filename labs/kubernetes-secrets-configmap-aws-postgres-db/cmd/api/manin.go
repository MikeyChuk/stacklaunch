package main

import (
	"log"

	"kubernetes-secrets-configmap/internal/api"
	"kubernetes-secrets-configmap/internal/db"
)

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	router := api.NewRouter(database)

	log.Println("API listening on port 8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
