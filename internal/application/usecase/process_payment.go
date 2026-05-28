package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"go.opentelemetry.io/otel"
)

type ProcessPayment struct {
	repo outgoing.PaymentRepositoryPort
}

func NewProcessPayment(repo outgoing.PaymentRepositoryPort) *ProcessPayment {
	return &ProcessPayment{repo: repo}
}

func (uc *ProcessPayment) Execute(ctx context.Context, payment *model.Payment) (*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "ProcessPayment.Execute")
	defer span.End()

	return uc.repo.Save(ctx, payment)
}
