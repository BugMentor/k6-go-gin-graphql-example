package outgoing

import (
	"context"
	"time"

	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
)

type PaymentRepositoryPort interface {
	Save(ctx context.Context, payment *model.Payment) (*model.Payment, error)
	SaveAll(ctx context.Context, payments []*model.Payment) error
	Update(ctx context.Context, payment *model.Payment) (*model.Payment, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error)
	FindByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status string, limit int) ([]*model.Payment, error)
	Search(ctx context.Context, minAmount, maxAmount *float64, currency, status string, page, size int) ([]*model.Payment, error)
	GetSummaryReport(ctx context.Context, startDate, endDate time.Time) (*model.PaymentSummary, error)

	TransferFunds(ctx context.Context, walletID, merchantID uuid.UUID, amount float64) (*model.Payment, error)
	TopUpWallet(ctx context.Context, walletID uuid.UUID, amount float64) (*model.Wallet, error)

	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	ListUsers(ctx context.Context) ([]*model.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	CreateMerchant(ctx context.Context, merchant *model.Merchant) (*model.Merchant, error)
	FindMerchantByID(ctx context.Context, id uuid.UUID) (*model.Merchant, error)
	ListMerchants(ctx context.Context) ([]*model.Merchant, error)
	DeleteMerchant(ctx context.Context, id uuid.UUID) error

	CreateWallet(ctx context.Context, wallet *model.Wallet) (*model.Wallet, error)
	FindWalletByID(ctx context.Context, id uuid.UUID) (*model.Wallet, error)
	FindWalletByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error)
	ListWallets(ctx context.Context) ([]*model.Wallet, error)
}
