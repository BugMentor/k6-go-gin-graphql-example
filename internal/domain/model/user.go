package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"fullName"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewUser(email, fullName, status string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if fullName == "" {
		return nil, ErrEmptyFullName
	}
	if status == "" {
		status = "ACTIVE"
	}
	return &User{
		ID:        uuid.New(),
		Email:     email,
		FullName:  fullName,
		Status:    status,
		CreatedAt: time.Now(),
	}, nil
}
