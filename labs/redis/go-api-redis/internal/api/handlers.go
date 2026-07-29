package api

import (
	"encoding/json"
	"net/http"

	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
	ctx := c.Request.Context()
	const cacheKey = "users:all"

	redisClient, redisAvailable := getRedisClient(c)
	log.Printf("Redis available in getUsers: %v", redisAvailable)

	// Step 1: Try Redis first.
	if redisAvailable {
		cachedUsers, err := redisClient.Get(ctx, cacheKey).Result()

		switch {
		case err == nil:
			var users []User

			if err := json.Unmarshal([]byte(cachedUsers), &users); err == nil {
				log.Println("Redis cache HIT for users:all")
				c.Header("X-Cache", "HIT")
				c.JSON(http.StatusOK, users)
				return
			}

			// Invalid cached JSON should not break the API.
			log.Printf("invalid Redis data for %s; deleting cached value", cacheKey)

			if err := redisClient.Del(ctx, cacheKey).Err(); err != nil {
				log.Printf("failed to delete invalid Redis key %s: %v", cacheKey, err)
			}

		case errors.Is(err, redis.Nil):
			log.Println("Redis cache MISS for users:all")

		default:
			// Redis failure should not stop the PostgreSQL request.
			log.Printf("Redis GET failed for %s: %v", cacheKey, err)
		}
	}

	// Step 2: Redis did not return usable data, so query PostgreSQL.
	db, ok := getDatabase(c)
	if !ok {
		return
	}

	rows, err := db.QueryContext(
		ctx,
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

	// Step 3: Save the PostgreSQL result in Redis.
	if redisAvailable {
		jsonData, err := json.Marshal(users)
		if err != nil {
			log.Printf("failed to serialise users for Redis: %v", err)
		} else {
			err = redisClient.Set(
				ctx,
				cacheKey,
				jsonData,
				5*time.Minute,
			).Err()

			if err != nil {
				log.Printf("Redis SET failed for %s: %v", cacheKey, err)
			} else {
				log.Printf("saved %s in Redis with a five-minute TTL", cacheKey)
			}
		}
	}

	c.Header("X-Cache", "MISS")
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
