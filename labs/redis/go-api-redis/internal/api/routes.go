package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func NewRouter(
	database *sql.DB,
	redisClient *redis.Client) *gin.Engine {
	router := gin.Default()

	router.Use(databaseMiddleware(database))
	router.Use(redisMiddleware(redisClient))

	router.GET("/health", healthCheck)
	router.GET("/users", getUsers)
	router.POST("/users", createUser)

	return router
}
