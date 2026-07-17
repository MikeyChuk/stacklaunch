package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

const databaseContextKey = "database"

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
