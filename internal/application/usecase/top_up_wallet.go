package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type TopUpWallet struct {
	repo outgoing.PaymentRepositoryPort
}

func NewTopUpWallet(repo outgoing.PaymentRepositoryPort) *TopUpWallet {
	return &TopUpWallet{repo: repo}
}

func (uc *TopUpWallet) Execute(ctx context.Context, walletID uuid.UUID, amount float64) (*model.Wallet, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "TopUpWallet.Execute")
	defer span.End()

	return uc.repo.TopUpWallet(ctx, walletID, amount)
}
