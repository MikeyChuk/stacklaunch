package models

type User struct {
	ID             uint
	Email          string
	HashedPassword string
}
