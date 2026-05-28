package model

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	MerchantID uuid.UUID `json:"merchantId"`
	WalletID   *uuid.UUID `json:"walletId,omitempty"`
	Amount     float64   `json:"amount"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Metadata   string    `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	Version    int64     `json:"version"`
	User       *User     `json:"user,omitempty"`
	Merchant   *Merchant `json:"merchant,omitempty"`
	Wallet     *Wallet   `json:"wallet,omitempty"`
}

func NewPayment(userID, merchantID uuid.UUID, walletID *uuid.UUID, amount float64, paymentType, status string) (*Payment, error) {
	if amount <= 0 {
		return nil, ErrPositiveAmountRequired
	}
	if paymentType == "" {
		paymentType = "DEBIT"
	}
	if status == "" {
		status = "PENDING"
	}
	return &Payment{
		ID:         uuid.New(),
		UserID:     userID,
		MerchantID: merchantID,
		WalletID:   walletID,
		Amount:     amount,
		Type:       paymentType,
		Status:     status,
		CreatedAt:  time.Now(),
		Version:    0,
	}, nil
}

func (p *Payment) Refund() error {
	if p.Status != "SUCCESS" {
		return ErrRefundNotAllowed
	}
	p.Status = "REFUNDED"
	return nil
}
