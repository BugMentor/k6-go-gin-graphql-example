package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Merchant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"apiKey"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewMerchant(name, apiKey string) (*Merchant, error) {
	if name == "" {
		return nil, errors.New("merchant name cannot be empty")
	}
	if apiKey == "" {
		return nil, errors.New("merchant api key cannot be empty")
	}
	return &Merchant{
		ID:        uuid.New(),
		Name:      name,
		APIKey:    apiKey,
		CreatedAt: time.Now(),
	}, nil
}
