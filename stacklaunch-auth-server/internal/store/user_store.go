package store

import (
	"errors"

	"github.com/michael/stacklaunch-auth-server/internal/models"
)

var users = []models.User{}
var nextUserID uint = 1

func CreateUser(email string, hashedPassword string) (models.User, error) {
	for _, user := range users {
		if user.Email == email {
			return models.User{}, errors.New("email already exists")
		}
	}

	user := models.User{
		ID:             nextUserID,
		Email:          email,
		HashedPassword: hashedPassword,
	}

	users = append(users, user)
	nextUserID++

	return user, nil
}

func FindUserByEmail(email string) (models.User, error) {
	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}

	return models.User{}, errors.New("user not found")
}
