package db

import "github.com/google/uuid"

func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
