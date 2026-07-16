package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michael/stacklaunch-auth-server/internal/auth"
	"github.com/michael/stacklaunch-auth-server/internal/dbsqlc"
)

type AuthHandler struct {
	database *sql.DB
	queries  *dbsqlc.Queries
}

func NewAuthHandler(
	database *sql.DB,
	queries *dbsqlc.Queries,
) *AuthHandler {
	return &AuthHandler{
		database: database,
		queries:  queries,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RegisterHandler creates a new user.
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email and password are required",
		})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	user, err := h.queries.CreateUser(
		c.Request.Context(),
		dbsqlc.CreateUserParams{
			Email:          req.Email,
			HashedPassword: hashedPassword,
		},
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "email already registered",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// LoginHandler verifies credentials and issues access and refresh tokens.
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email and password are required",
		})
		return
	}

	user, err := h.queries.GetUserByEmail(
		c.Request.Context(),
		req.Email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch user",
		})
		return
	}

	if !auth.CheckPassword(req.Password, user.HashedPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid email or password",
		})
		return
	}

	accessToken, err := auth.GenerateToken(
		uint(user.ID),
		user.Email,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate access token",
		})
		return
	}

	refreshToken, err := createRefreshToken(
		c.Request.Context(),
		h.queries,
		user.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// RefreshHandler revokes the old refresh token and issues a new token pair.
func (h *AuthHandler) RefreshHandler(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refresh token is required",
		})
		return
	}

	oldTokenHash := auth.HashRefreshToken(req.RefreshToken)

	tx, err := h.database.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start token rotation",
		})
		return
	}

	// Rollback does nothing if the transaction has already been committed.
	defer tx.Rollback()

	txQueries := h.queries.WithTx(tx)

	oldToken, err := txQueries.ConsumeRefreshToken(
		c.Request.Context(),
		oldTokenHash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid, expired or revoked refresh token",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to validate refresh token",
		})
		return
	}

	user, err := txQueries.GetUserByID(
		c.Request.Context(),
		oldToken.UserID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user no longer exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch user",
		})
		return
	}

	newRefreshToken, err := createRefreshToken(
		c.Request.Context(),
		txQueries,
		user.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to rotate refresh token",
		})
		return
	}

	newAccessToken, err := auth.GenerateToken(
		uint(user.ID),
		user.Email,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate access token",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to complete token rotation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// LogoutHandler revokes the supplied refresh token.
func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refresh token is required",
		})
		return
	}

	tokenHash := auth.HashRefreshToken(req.RefreshToken)

	_, err := h.queries.RevokeRefreshToken(
		c.Request.Context(),
		tokenHash,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to log out",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// MeHandler returns information placed in the Gin context by AuthMiddleware.
func MeHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "email not found in token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "authenticated user",
		"user": gin.H{
			"id":    userID,
			"email": email,
		},
	})
}

// createRefreshToken generates a random refresh token,
// stores its hash in PostgreSQL and returns the original token.
func createRefreshToken(
	ctx context.Context,
	queries *dbsqlc.Queries,
	userID int64,
) (string, error) {
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	tokenHash := auth.HashRefreshToken(refreshToken)

	_, err = queries.CreateRefreshToken(
		ctx,
		dbsqlc.CreateRefreshTokenParams{
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		},
	)
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}
