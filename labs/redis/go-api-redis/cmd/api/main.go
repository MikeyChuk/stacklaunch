package main

import (
	"context"
	"log"

	"kubernetes-secrets-configmap/internal/api"
	"kubernetes-secrets-configmap/internal/cache"
	"kubernetes-secrets-configmap/internal/db"
)

func main() {

	ctx := context.Background()

	redisClient, err := cache.NewRedisClient(ctx)
	if err != nil {
		log.Printf("Redis unavailable; API will continue without caching: %v", err)
	} else {
		log.Println("successfully connected to Redis")

		defer func() {
			if err := redisClient.Close(); err != nil {
				log.Printf("failed to close Redis client: %v", err)
			}
		}()
	}

	log.Println("1. Application starting")
	log.Println("2. Connecting to PostgreSQL")
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	log.Println("3. Database connected")

	router := api.NewRouter(database, redisClient)
	log.Println("API listening on port 8080")

	log.Println("4. API listening on port 8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
