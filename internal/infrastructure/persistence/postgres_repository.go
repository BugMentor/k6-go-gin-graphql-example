package persistence

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Save(ctx context.Context, payment *model.Payment) (*model.Payment, error) {
	metadataJSON, _ := json.Marshal(struct{}{})

	err := r.pool.QueryRow(ctx,
		`INSERT INTO payments (id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, version = payment.version + 1
		 RETURNING id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version`,
		payment.ID, payment.UserID, payment.MerchantID, payment.WalletID, payment.Amount,
		payment.Type, payment.Status, metadataJSON, payment.CreatedAt, payment.Version,
	).Scan(&payment.ID, &payment.UserID, &payment.MerchantID, &payment.WalletID,
		&payment.Amount, &payment.Type, &payment.Status, &metadataJSON, &payment.CreatedAt, &payment.Version)

	if err != nil {
		return nil, err
	}
	return payment, nil
}

func (r *PostgresRepository) SaveAll(ctx context.Context, payments []*model.Payment) error {
	batch := &pgx.Batch{}
	metadataJSON, _ := json.Marshal(struct{}{})

	for _, p := range payments {
		batch.Queue(
			`INSERT INTO payments (id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			p.ID, p.UserID, p.MerchantID, p.WalletID, p.Amount,
			p.Type, p.Status, metadataJSON, p.CreatedAt, p.Version,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range payments {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, payment *model.Payment) (*model.Payment, error) {
	metadataJSON, _ := json.Marshal(struct{}{})
	var version int64

	err := r.pool.QueryRow(ctx,
		`UPDATE payments SET status = $1, metadata = $2, version = version + 1
		 WHERE id = $3 AND version = $4
		 RETURNING id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version`,
		payment.Status, metadataJSON, payment.ID, payment.Version,
	).Scan(&payment.ID, &payment.UserID, &payment.MerchantID, &payment.WalletID,
		&payment.Amount, &payment.Type, &payment.Status, &metadataJSON, &payment.CreatedAt, &version)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrVersionConflict
		}
		return nil, err
	}

	payment.Version = version
	return payment, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	p := &model.Payment{}
	var metadataJSON []byte
	var walletID *uuid.UUID

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version
		 FROM payments WHERE id = $1`, id,
	).Scan(&p.ID, &p.UserID, &p.MerchantID, &walletID, &p.Amount,
		&p.Type, &p.Status, &metadataJSON, &p.CreatedAt, &p.Version)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	p.WalletID = walletID
	return p, nil
}

func (r *PostgresRepository) FindByUserIDAndStatus(ctx context.Context, userID uuid.UUID, status string, limit int) ([]*model.Payment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version
		 FROM payments WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3`,
		userID, status, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPayments(rows)
}

func (r *PostgresRepository) Search(ctx context.Context, minAmount, maxAmount *float64, currency, status string, page, size int) ([]*model.Payment, error) {
	query := `SELECT id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version
			  FROM payments WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if minAmount != nil {
		query += ` AND amount >= $` + itoa(argIdx)
		args = append(args, *minAmount)
		argIdx++
	}
	if maxAmount != nil {
		query += ` AND amount <= $` + itoa(argIdx)
		args = append(args, *maxAmount)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + itoa(argIdx)
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY created_at DESC`
	query += ` LIMIT $` + itoa(argIdx)
	args = append(args, size)
	argIdx++
	query += ` OFFSET $` + itoa(argIdx)
	args = append(args, page*size)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPayments(rows)
}

func (r *PostgresRepository) GetSummaryReport(ctx context.Context, startDate, endDate time.Time) (*model.PaymentSummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COALESCE(SUM(amount), 0) as total, COUNT(*) as cnt
		 FROM payments WHERE created_at BETWEEN $1 AND $2 GROUP BY status`,
		startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := model.NewPaymentSummary()
	for rows.Next() {
		var status string
		var total float64
		var cnt int
		if err := rows.Scan(&status, &total, &cnt); err != nil {
			return nil, err
		}
		summary.TotalsByStatus[status] = total
		summary.TotalCount += cnt
		summary.TotalAmount += total
	}

	return summary, nil
}

func (r *PostgresRepository) TransferFunds(ctx context.Context, walletID, merchantID uuid.UUID, amount float64) (*model.Payment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID uuid.UUID
	var balance float64
	var version int64
	err = tx.QueryRow(ctx,
		`SELECT user_id, balance, version FROM wallets WHERE id = $1 FOR UPDATE`, walletID,
	).Scan(&userID, &balance, &version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	if balance < amount {
		return nil, model.ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx,
		`UPDATE wallets SET balance = balance - $1, version = version + 1 WHERE id = $2 AND version = $3`,
		amount, walletID, version,
	)
	if err != nil {
		return nil, err
	}

	payment := &model.Payment{
		ID:         uuid.New(),
		UserID:     userID,
		MerchantID: merchantID,
		WalletID:   &walletID,
		Amount:     amount,
		Type:       "WALLET_TRANSFER",
		Status:     "SUCCESS",
		CreatedAt:  time.Now(),
		Version:    0,
	}

	metadataJSON, _ := json.Marshal(struct{}{})
	_, err = tx.Exec(ctx,
		`INSERT INTO payments (id, user_id, merchant_id, wallet_id, amount, type, status, metadata, created_at, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		payment.ID, payment.UserID, payment.MerchantID, payment.WalletID, payment.Amount,
		payment.Type, payment.Status, metadataJSON, payment.CreatedAt, payment.Version,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return payment, nil
}

func (r *PostgresRepository) TopUpWallet(ctx context.Context, walletID uuid.UUID, amount float64) (*model.Wallet, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var version int64
	err = tx.QueryRow(ctx,
		`SELECT version FROM wallets WHERE id = $1 FOR UPDATE`, walletID,
	).Scan(&version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	wallet := &model.Wallet{}
	err = tx.QueryRow(ctx,
		`UPDATE wallets SET balance = balance + $1, version = version + 1
		 WHERE id = $2 AND version = $3
		 RETURNING id, user_id, balance, currency, version, created_at`,
		amount, walletID, version,
	).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.Currency, &wallet.Version, &wallet.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrVersionConflict
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return wallet, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (id, email, full_name, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, email, full_name, status, created_at`,
		user.ID, user.Email, user.FullName, user.Status, user.CreatedAt,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, full_name, status, created_at FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Status, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, full_name, status, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateMerchant(ctx context.Context, merchant *model.Merchant) (*model.Merchant, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO merchants (id, name, api_key, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, api_key, created_at`,
		merchant.ID, merchant.Name, merchant.APIKey, merchant.CreatedAt,
	).Scan(&merchant.ID, &merchant.Name, &merchant.APIKey, &merchant.CreatedAt)
	if err != nil {
		return nil, err
	}
	return merchant, nil
}

func (r *PostgresRepository) FindMerchantByID(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	merchant := &model.Merchant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, api_key, created_at FROM merchants WHERE id = $1`, id,
	).Scan(&merchant.ID, &merchant.Name, &merchant.APIKey, &merchant.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return merchant, nil
}

func (r *PostgresRepository) ListMerchants(ctx context.Context) ([]*model.Merchant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, api_key, created_at FROM merchants ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []*model.Merchant
	for rows.Next() {
		m := &model.Merchant{}
		if err := rows.Scan(&m.ID, &m.Name, &m.APIKey, &m.CreatedAt); err != nil {
			return nil, err
		}
		merchants = append(merchants, m)
	}
	return merchants, nil
}

func (r *PostgresRepository) DeleteMerchant(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM merchants WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateWallet(ctx context.Context, wallet *model.Wallet) (*model.Wallet, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO wallets (id, user_id, balance, currency, version, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, balance, currency, version, created_at`,
		wallet.ID, wallet.UserID, wallet.Balance, wallet.Currency, wallet.Version, wallet.CreatedAt,
	).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.Currency, &wallet.Version, &wallet.CreatedAt)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

func (r *PostgresRepository) FindWalletByID(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	wallet := &model.Wallet{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, balance, currency, version, created_at FROM wallets WHERE id = $1`, id,
	).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.Currency, &wallet.Version, &wallet.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return wallet, nil
}

func (r *PostgresRepository) FindWalletByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	wallet := &model.Wallet{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, balance, currency, version, created_at FROM wallets WHERE user_id = $1`, userID,
	).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.Currency, &wallet.Version, &wallet.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return wallet, nil
}

func (r *PostgresRepository) ListWallets(ctx context.Context) ([]*model.Wallet, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, balance, currency, version, created_at FROM wallets ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []*model.Wallet
	for rows.Next() {
		w := &model.Wallet{}
		if err := rows.Scan(&w.ID, &w.UserID, &w.Balance, &w.Currency, &w.Version, &w.CreatedAt); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}

func scanPayments(rows pgx.Rows) ([]*model.Payment, error) {
	var payments []*model.Payment
	for rows.Next() {
		p := &model.Payment{}
		var metadataJSON []byte
		var walletID *uuid.UUID
		if err := rows.Scan(&p.ID, &p.UserID, &p.MerchantID, &walletID, &p.Amount,
			&p.Type, &p.Status, &metadataJSON, &p.CreatedAt, &p.Version); err != nil {
			return nil, err
		}
		p.WalletID = walletID
		payments = append(payments, p)
	}
	return payments, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
