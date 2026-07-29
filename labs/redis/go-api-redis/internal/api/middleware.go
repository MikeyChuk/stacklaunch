package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const databaseContextKey = "database"
const redisContextKey = "redis"

func databaseMiddleware(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(databaseContextKey, database)
		c.Next()
	}
}

func getDatabase(c *gin.Context) (*sql.DB, bool) {
	value, exists := c.Get(databaseContextKey)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "database connection is unavailable",
		})
		return nil, false
	}

	database, ok := value.(*sql.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid database connection",
		})
		return nil, false
	}

	return database, true
}

func redisMiddleware(client *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client != nil {
			c.Set(redisContextKey, client)
		}
		c.Next()
	}
}

func getRedisClient(c *gin.Context) (*redis.Client, bool) {
	value, exists := c.Get(redisContextKey)
	if !exists {
		return nil, false
	}

	client, ok := value.(*redis.Client)
	return client, ok
}
