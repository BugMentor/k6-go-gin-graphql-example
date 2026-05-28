package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type WalletTransfer struct {
	repo outgoing.PaymentRepositoryPort
}

func NewWalletTransfer(repo outgoing.PaymentRepositoryPort) *WalletTransfer {
	return &WalletTransfer{repo: repo}
}

func (uc *WalletTransfer) Execute(ctx context.Context, walletID, merchantID uuid.UUID, amount float64) (*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "WalletTransfer.Execute")
	defer span.End()

	return uc.repo.TransferFunds(ctx, walletID, merchantID, amount)
}
