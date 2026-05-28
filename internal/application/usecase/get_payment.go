package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type GetPayment struct {
	repo outgoing.PaymentRepositoryPort
}

func NewGetPayment(repo outgoing.PaymentRepositoryPort) *GetPayment {
	return &GetPayment{repo: repo}
}

func (uc *GetPayment) Execute(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "GetPayment.Execute")
	defer span.End()

	return uc.repo.FindByID(ctx, id)
}
