package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/domain/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RegisterRoutes(router *gin.Engine, repo outgoing.PaymentRepositoryPort) {
	v1 := router.Group("/v1")
	{
		v1.POST("/users", createUser(repo))
		v1.GET("/users", listUsers(repo))
		v1.GET("/users/:id", getUser(repo))
		v1.DELETE("/users/:id", deleteUser(repo))

		v1.POST("/merchants", createMerchant(repo))
		v1.GET("/merchants", listMerchants(repo))
		v1.GET("/merchants/:id", getMerchant(repo))
		v1.DELETE("/merchants/:id", deleteMerchant(repo))

		v1.POST("/wallets", createWallet(repo))
		v1.GET("/wallets", listWallets(repo))
		v1.GET("/wallets/:id", getWallet(repo))
		v1.GET("/wallets/user/:userId", getWalletByUserID(repo))

		v1.POST("/payments", createPayment(repo))
		v1.POST("/payments/batch", createBatchPayments(repo))
		v1.POST("/payments/wallet-transfer", walletTransfer(repo))
		v1.POST("/payments/wallets/:id/topup", topUpWallet(repo))
		v1.PUT("/payments/:id/refund", refundPayment(repo))
		v1.GET("/payments/:id", getPayment(repo))
		v1.GET("/payments/user/:userId", listUserPayments(repo))
		v1.GET("/payments/search", searchPayments(repo))
		v1.GET("/payments/reports/summary", getPaymentSummary(repo))
	}
}

func createUser(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			FullName string `json:"fullName"`
			Status   string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		user, err := model.NewUser(req.Email, req.FullName, req.Status)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		created, err := repo.CreateUser(c.Request.Context(), user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func listUsers(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := repo.ListUsers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

func getUser(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		user, err := repo.FindUserByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func deleteUser(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		if err := repo.DeleteUser(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func createMerchant(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name   string `json:"name"`
			APIKey string `json:"apiKey"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		merchant, err := model.NewMerchant(req.Name, req.APIKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		created, err := repo.CreateMerchant(c.Request.Context(), merchant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func listMerchants(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		merchants, err := repo.ListMerchants(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, merchants)
	}
}

func getMerchant(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		merchant, err := repo.FindMerchantByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, merchant)
	}
}

func deleteMerchant(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		if err := repo.DeleteMerchant(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func createWallet(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID   string  `json:"userId"`
			Balance  float64 `json:"balance"`
			Currency string  `json:"currency"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID"))
			return
		}
		wallet, err := model.NewWallet(userID, req.Balance, req.Currency)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		created, err := repo.CreateWallet(c.Request.Context(), wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func listWallets(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		wallets, err := repo.ListWallets(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, wallets)
	}
}

func getWallet(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		wallet, err := repo.FindWalletByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, wallet)
	}
}

func getWalletByUserID(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("userId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID"))
			return
		}
		wallet, err := repo.FindWalletByUserID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, wallet)
	}
}

func createPayment(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID     string  `json:"userId"`
			MerchantID string  `json:"merchantId"`
			WalletID   *string `json:"walletId"`
			Amount     float64 `json:"amount"`
			Type       string  `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		userID, _ := uuid.Parse(req.UserID)
		merchantID, _ := uuid.Parse(req.MerchantID)
		var walletID *uuid.UUID
		if req.WalletID != nil {
			parsed := uuid.MustParse(*req.WalletID)
			walletID = &parsed
		}
		paymentType := req.Type
		if paymentType == "" {
			paymentType = "DEBIT"
		}
		payment, err := model.NewPayment(userID, merchantID, walletID, req.Amount, paymentType, "SUCCESS")
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		created, err := repo.Save(c.Request.Context(), payment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func createBatchPayments(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []struct {
			UserID     string  `json:"userId"`
			MerchantID string  `json:"merchantId"`
			Amount     float64 `json:"amount"`
			Type       string  `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		var payments []*model.Payment
		for _, item := range req {
			userID, _ := uuid.Parse(item.UserID)
			merchantID, _ := uuid.Parse(item.MerchantID)
			paymentType := item.Type
			if paymentType == "" {
				paymentType = "DEBIT"
			}
			payment, err := model.NewPayment(userID, merchantID, nil, item.Amount, paymentType, "SUCCESS")
			if err != nil {
				c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
				return
			}
			payments = append(payments, payment)
		}
		if err := repo.SaveAll(c.Request.Context(), payments); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusCreated, gin.H{"count": len(payments), "success": true})
	}
}

func walletTransfer(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			WalletID   string  `json:"walletId"`
			MerchantID string  `json:"merchantId"`
			Amount     float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		walletID, _ := uuid.Parse(req.WalletID)
		merchantID, _ := uuid.Parse(req.MerchantID)
		payment, err := repo.TransferFunds(c.Request.Context(), walletID, merchantID, req.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, payment)
	}
}

func topUpWallet(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid wallet ID"))
			return
		}
		var req struct {
			Amount float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid request body"))
			return
		}
		wallet, err := repo.TopUpWallet(c.Request.Context(), id, req.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, wallet)
	}
}

func refundPayment(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		payment, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		if err := payment.Refund(); err != nil {
			c.JSON(http.StatusConflict, errorResponse(err.Error()))
			return
		}
		updated, err := repo.Update(c.Request.Context(), payment)
		if err != nil {
			c.JSON(http.StatusConflict, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, updated)
	}
}

func getPayment(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid ID format"))
			return
		}
		payment, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, payment)
	}
}

func listUserPayments(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("userId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID"))
			return
		}
		status := c.DefaultQuery("status", "SUCCESS")
		limit := 10
		payments, err := repo.FindByUserIDAndStatus(c.Request.Context(), userID, status, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, payments)
	}
}

func searchPayments(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "")
		currency := c.DefaultQuery("currency", "")
		page := 0
		size := 10
		var minAmount, maxAmount *float64
		if v := c.Query("minAmount"); v != "" {
			f, _ := parseFloat(v)
			minAmount = &f
		}
		if v := c.Query("maxAmount"); v != "" {
			f, _ := parseFloat(v)
			maxAmount = &f
		}
		payments, err := repo.Search(c.Request.Context(), minAmount, maxAmount, currency, status, page, size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, payments)
	}
}

func getPaymentSummary(repo outgoing.PaymentRepositoryPort) gin.HandlerFunc {
	return func(c *gin.Context) {
		startDate, _ := time.Parse(time.RFC3339, c.Query("startDate"))
		endDate, _ := time.Parse(time.RFC3339, c.Query("endDate"))
		summary, err := repo.GetSummaryReport(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(err.Error()))
			return
		}
		c.JSON(http.StatusOK, summary)
	}
}

func errorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    http.StatusBadRequest,
		"error":     "Bad Request",
		"message":   message,
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
