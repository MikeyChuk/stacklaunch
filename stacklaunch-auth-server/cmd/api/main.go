package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	appdb "github.com/michael/stacklaunch-auth-server/internal/db"
	"github.com/michael/stacklaunch-auth-server/internal/dbsqlc"
	"github.com/michael/stacklaunch-auth-server/internal/handlers"
	"github.com/michael/stacklaunch-auth-server/internal/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	database, err := appdb.Connect()
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer database.Close()

	queries := dbsqlc.New(database)
	authHandler := handlers.NewAuthHandler(database, queries)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/register", authHandler.RegisterHandler)
	r.POST("/login", authHandler.LoginHandler)
	r.POST("/refresh", authHandler.RefreshHandler)
	r.POST("/logout", authHandler.LogoutHandler)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	protected.GET("/me", handlers.MeHandler)

	r.Run(":8080")
}
