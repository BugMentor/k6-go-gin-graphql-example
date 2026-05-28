package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"go.opentelemetry.io/otel"
)

type SearchPayments struct {
	repo outgoing.PaymentRepositoryPort
}

func NewSearchPayments(repo outgoing.PaymentRepositoryPort) *SearchPayments {
	return &SearchPayments{repo: repo}
}

func (uc *SearchPayments) Execute(ctx context.Context, minAmount, maxAmount *float64, currency, status string, page, size int) ([]*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "SearchPayments.Execute")
	defer span.End()

	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = 10
	}

	return uc.repo.Search(ctx, minAmount, maxAmount, currency, status, page, size)
}
