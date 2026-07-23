package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func createUser(c *gin.Context) {
	db, ok := getDatabase(c)
	if !ok {
		return
	}

	var input CreateUserRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "a valid name and email are required",
		})
		return
	}

	var user User

	err := db.QueryRowContext(
		c.Request.Context(),
		`
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email, created_at
		`,
		input.Name,
		input.Email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func getUsers(c *gin.Context) {
	db, ok := getDatabase(c)
	if !ok {
		return
	}

	rows, err := db.QueryContext(
		c.Request.Context(),
		`
		SELECT id, name, email, created_at
		FROM users
		ORDER BY id
		`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve users",
		})
		return
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var user User

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to read user data",
			})
			return
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed while reading users",
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

func healthCheck(c *gin.Context) {
	db, ok := getDatabase(c)
	if !ok {
		return
	}

	if err := db.PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "disconnected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
		"version":  "1.1.0",
	})
}
