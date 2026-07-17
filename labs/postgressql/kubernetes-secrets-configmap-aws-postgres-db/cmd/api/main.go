package main

import (
	"log"

	"kubernetes-secrets-configmap/internal/api"
	"kubernetes-secrets-configmap/internal/db"
)

func main() {
	log.Println("1. Application starting")
	log.Println("2. Connecting to PostgreSQL")
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	log.Println("3. Database connected")

	router := api.NewRouter(database)

	log.Println("API listening on port 8080")

	log.Println("4. API listening on port 8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
