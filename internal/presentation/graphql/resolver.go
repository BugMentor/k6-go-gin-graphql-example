package graphql

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
)

func resolvePayment(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return getPaymentUC.Execute(context.Background(), id)
}

func resolvePayments(p graphql.ResolveParams) (interface{}, error) {
	userID := p.Args["userId"].(string)
	status := ""
	if s, ok := p.Args["status"].(string); ok {
		status = s
	}
	limit := 10
	if l, ok := p.Args["limit"].(int); ok {
		limit = l
	}
	return listUserPaymentsUC.Execute(context.Background(), userID, status, limit)
}

func resolveSearchPayments(p graphql.ResolveParams) (interface{}, error) {
	var minAmount, maxAmount *float64
	if v, ok := p.Args["minAmount"].(float64); ok {
		minAmount = &v
	}
	if v, ok := p.Args["maxAmount"].(float64); ok {
		maxAmount = &v
	}
	currency := ""
	if v, ok := p.Args["currency"].(string); ok {
		currency = v
	}
	status := ""
	if v, ok := p.Args["status"].(string); ok {
		status = v
	}
	page := 0
	if v, ok := p.Args["page"].(int); ok {
		page = v
	}
	size := 10
	if v, ok := p.Args["size"].(int); ok {
		size = v
	}
	return searchPaymentsUC.Execute(context.Background(), minAmount, maxAmount, currency, status, page, size)
}

func resolvePaymentSummary(p graphql.ResolveParams) (interface{}, error) {
	startDate, err := time.Parse(time.RFC3339, p.Args["startDate"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid startDate: %w", err)
	}
	endDate, err := time.Parse(time.RFC3339, p.Args["endDate"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid endDate: %w", err)
	}
	summary, err := getPaymentSummaryUC.Execute(context.Background(), startDate, endDate)
	if err != nil {
		return nil, err
	}

	var totals []map[string]interface{}
	for status, total := range summary.TotalsByStatus {
		totals = append(totals, map[string]interface{}{
			"status": status,
			"total":  math.Round(total*100) / 100,
		})
	}

	return map[string]interface{}{
		"totalsByStatus": totals,
		"totalCount":     summary.TotalCount,
		"totalAmount":    math.Round(summary.TotalAmount*100) / 100,
	}, nil
}

func resolveProcessPayment(p graphql.ResolveParams) (interface{}, error) {
	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid input")
	}

	userID, _ := uuid.Parse(input["userId"].(string))
	merchantID, _ := uuid.Parse(input["merchantId"].(string))
	amount, _ := input["amount"].(float64)
	paymentType := "DEBIT"
	if t, ok := input["type"].(string); ok {
		paymentType = t
	}

	payment, err := model.NewPayment(userID, merchantID, nil, amount, paymentType, "PENDING")
	if err != nil {
		return nil, err
	}

	return processPaymentUC.Execute(context.Background(), payment)
}

func resolveWalletTransfer(p graphql.ResolveParams) (interface{}, error) {
	walletID, _ := uuid.Parse(p.Args["walletId"].(string))
	merchantID, _ := uuid.Parse(p.Args["merchantId"].(string))
	amount, _ := p.Args["amount"].(float64)

	return walletTransferUC.Execute(context.Background(), walletID, merchantID, amount)
}

func resolveTopUpWallet(p graphql.ResolveParams) (interface{}, error) {
	walletID, _ := uuid.Parse(p.Args["walletId"].(string))
	amount, _ := p.Args["amount"].(float64)

	return topUpWalletUC.Execute(context.Background(), walletID, amount)
}

func resolveProcessBatchPayments(p graphql.ResolveParams) (interface{}, error) {
	inputs, ok := p.Args["payments"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payments list")
	}

	var payments []*model.Payment
	for _, item := range inputs {
		input, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		userID, _ := uuid.Parse(input["userId"].(string))
		merchantID, _ := uuid.Parse(input["merchantId"].(string))
		amount, _ := input["amount"].(float64)
		paymentType := "DEBIT"
		if t, ok := input["type"].(string); ok {
			paymentType = t
		}
		payment, err := model.NewPayment(userID, merchantID, nil, amount, paymentType, "PENDING")
		if err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	return true, processBatchPaymentsUC.Execute(context.Background(), payments)
}

func resolveRefundPayment(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return refundPaymentUC.Execute(context.Background(), id)
}

func resolveUser(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return repo.FindUserByID(context.Background(), id)
}

func resolveUsers(p graphql.ResolveParams) (interface{}, error) {
	return repo.ListUsers(context.Background())
}

func resolveCreateUser(p graphql.ResolveParams) (interface{}, error) {
	email := p.Args["email"].(string)
	fullName := p.Args["fullName"].(string)
	status := "ACTIVE"
	if s, ok := p.Args["status"].(string); ok {
		status = s
	}
	user, err := model.NewUser(email, fullName, status)
	if err != nil {
		return nil, err
	}
	return repo.CreateUser(context.Background(), user)
}

func resolveDeleteUser(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return true, repo.DeleteUser(context.Background(), id)
}

func resolveMerchant(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return repo.FindMerchantByID(context.Background(), id)
}

func resolveMerchants(p graphql.ResolveParams) (interface{}, error) {
	return repo.ListMerchants(context.Background())
}

func resolveCreateMerchant(p graphql.ResolveParams) (interface{}, error) {
	name := p.Args["name"].(string)
	apiKey := p.Args["apiKey"].(string)
	merchant, err := model.NewMerchant(name, apiKey)
	if err != nil {
		return nil, err
	}
	return repo.CreateMerchant(context.Background(), merchant)
}

func resolveDeleteMerchant(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return true, repo.DeleteMerchant(context.Background(), id)
}

func resolveWallet(p graphql.ResolveParams) (interface{}, error) {
	id, err := uuid.Parse(p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return repo.FindWalletByID(context.Background(), id)
}

func resolveWallets(p graphql.ResolveParams) (interface{}, error) {
	return repo.ListWallets(context.Background())
}

func resolveWalletByUserID(p graphql.ResolveParams) (interface{}, error) {
	userID, err := uuid.Parse(p.Args["userId"].(string))
	if err != nil {
		return nil, err
	}
	return repo.FindWalletByUserID(context.Background(), userID)
}

func resolveCreateWallet(p graphql.ResolveParams) (interface{}, error) {
	userID, err := uuid.Parse(p.Args["userId"].(string))
	if err != nil {
		return nil, err
	}
	currency := "USD"
	if c, ok := p.Args["currency"].(string); ok && c != "" {
		currency = c
	}
	balance := 0.0
	if b, ok := p.Args["balance"].(float64); ok {
		balance = b
	}
	wallet, err := model.NewWallet(userID, balance, currency)
	if err != nil {
		return nil, err
	}
	return repo.CreateWallet(context.Background(), wallet)
}
