package model

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	User      *User     `json:"user,omitempty"`
}

func NewWallet(userID uuid.UUID, balance float64, currency string) (*Wallet, error) {
	if balance < 0 {
		return nil, ErrInsufficientFunds
	}
	if currency == "" {
		currency = "USD"
	}
	return &Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Balance:   balance,
		Currency:  currency,
		Version:   0,
		CreatedAt: time.Now(),
	}, nil
}

func (w *Wallet) Debit(amount float64) error {
	if amount <= 0 {
		return ErrPositiveAmountRequired
	}
	if w.Balance < amount {
		return ErrInsufficientFunds
	}
	w.Balance -= amount
	w.Version++
	return nil
}

func (w *Wallet) TopUp(amount float64) error {
	if amount <= 0 {
		return ErrPositiveAmountRequired
	}
	w.Balance += amount
	w.Version++
	return nil
}
