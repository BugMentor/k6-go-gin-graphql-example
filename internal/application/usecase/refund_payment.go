package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type RefundPayment struct {
	repo outgoing.PaymentRepositoryPort
}

func NewRefundPayment(repo outgoing.PaymentRepositoryPort) *RefundPayment {
	return &RefundPayment{repo: repo}
}

func (uc *RefundPayment) Execute(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "RefundPayment.Execute")
	defer span.End()

	payment, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := payment.Refund(); err != nil {
		return nil, err
	}

	return uc.repo.Update(ctx, payment)
}
