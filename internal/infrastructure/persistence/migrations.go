package persistence

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			full_name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS merchants (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			api_key VARCHAR(512) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS wallets (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL UNIQUE REFERENCES users(id),
			balance DECIMAL(16,2) NOT NULL DEFAULT 0.00,
			currency VARCHAR(3) NOT NULL DEFAULT 'USD',
			version BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			merchant_id UUID NOT NULL,
			wallet_id UUID,
			amount DECIMAL(16,2) NOT NULL,
			type VARCHAR(50) NOT NULL DEFAULT 'DEBIT',
			status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			version BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_user_status ON payments(user_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_merchant_id ON payments(merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_wallet_id ON payments(wallet_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_amount ON payments(amount)`,
		`CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id)`,
	}

	for i, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration V%d failed: %w", i+1, err)
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}
