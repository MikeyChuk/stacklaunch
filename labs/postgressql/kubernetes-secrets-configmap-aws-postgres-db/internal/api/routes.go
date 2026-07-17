package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func NewRouter(database *sql.DB) *gin.Engine {
	router := gin.Default()

	router.Use(databaseMiddleware(database))

	router.GET("/health", healthCheck)
	router.GET("/users", getUsers)
	router.POST("/users", createUser)

	return router
}
