package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"go.opentelemetry.io/otel"
)

type ProcessBatchPayments struct {
	repo outgoing.PaymentRepositoryPort
}

func NewProcessBatchPayments(repo outgoing.PaymentRepositoryPort) *ProcessBatchPayments {
	return &ProcessBatchPayments{repo: repo}
}

func (uc *ProcessBatchPayments) Execute(ctx context.Context, payments []*model.Payment) error {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "ProcessBatchPayments.Execute")
	defer span.End()

	return uc.repo.SaveAll(ctx, payments)
}
