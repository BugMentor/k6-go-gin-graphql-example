package usecase

import (
	"context"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type ListUserPayments struct {
	repo outgoing.PaymentRepositoryPort
}

func NewListUserPayments(repo outgoing.PaymentRepositoryPort) *ListUserPayments {
	return &ListUserPayments{repo: repo}
}

func (uc *ListUserPayments) Execute(ctx context.Context, userIDStr, status string, limit int) ([]*model.Payment, error) {
	ctx, span := otel.Tracer("payment-service").Start(ctx, "ListUserPayments.Execute")
	defer span.End()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	if status == "" {
		status = "SUCCESS"
	}
	if limit <= 0 {
		limit = 10
	}

	return uc.repo.FindByUserIDAndStatus(ctx, userID, status, limit)
}
