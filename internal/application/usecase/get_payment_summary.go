package usecase

import (
	"context"
	"time"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"go.opentelemetry.io/otel"
)

type GetPaymentSummary struct {
	repo outgoing.PaymentRepositoryPort
}

func NewGetPaymentSummary(repo outgoing.PaymentRepositoryPort) *GetPaymentSummary {
	return &GetPaymentSummary{repo: repo}
}

func (uc *GetPaymentSummary) Execute(ctx context.Context, startDate, endDate time.Time) (*model.PaymentSummary, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "GetPaymentSummary.Execute")
	defer span.End()

	return uc.repo.GetSummaryReport(ctx, startDate, endDate)
}
