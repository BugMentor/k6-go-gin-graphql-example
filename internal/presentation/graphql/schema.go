package graphql

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	"github.com/enterprise/payment-service/internal/application/port/outgoing"
	"github.com/enterprise/payment-service/internal/application/usecase"
	"github.com/gin-gonic/gin"
)

type UseCases struct {
	ProcessPayment       *usecase.ProcessPayment
	ProcessBatchPayments *usecase.ProcessBatchPayments
	RefundPayment        *usecase.RefundPayment
	GetPayment           *usecase.GetPayment
	ListUserPayments     *usecase.ListUserPayments
	SearchPayments       *usecase.SearchPayments
	GetPaymentSummary    *usecase.GetPaymentSummary
	WalletTransfer       *usecase.WalletTransfer
	TopUpWallet          *usecase.TopUpWallet
}

var (
	processPaymentUC       *usecase.ProcessPayment
	processBatchPaymentsUC *usecase.ProcessBatchPayments
	refundPaymentUC        *usecase.RefundPayment
	getPaymentUC           *usecase.GetPayment
	listUserPaymentsUC     *usecase.ListUserPayments
	searchPaymentsUC       *usecase.SearchPayments
	getPaymentSummaryUC    *usecase.GetPaymentSummary
	walletTransferUC       *usecase.WalletTransfer
	topUpWalletUC          *usecase.TopUpWallet
	repo                   outgoing.PaymentRepositoryPort
)

func RegisterRoutes(router *gin.Engine, ucs UseCases, r outgoing.PaymentRepositoryPort) {
	processPaymentUC = ucs.ProcessPayment
	processBatchPaymentsUC = ucs.ProcessBatchPayments
	refundPaymentUC = ucs.RefundPayment
	getPaymentUC = ucs.GetPayment
	listUserPaymentsUC = ucs.ListUserPayments
	searchPaymentsUC = ucs.SearchPayments
	getPaymentSummaryUC = ucs.GetPaymentSummary
	walletTransferUC = ucs.WalletTransfer
	topUpWalletUC = ucs.TopUpWallet
	repo = r

	schema, err := buildSchema()
	if err != nil {
		panic(err)
	}

	h := handler.New(&handler.Config{
		Schema:     &schema,
		Pretty:     true,
		GraphiQL:   true,
		Playground: true,
	})

	router.POST("/graphql", func(c *gin.Context) {
		h.ContextHandler(c.Request.Context(), c.Writer, c.Request)
	})

	router.GET("/graphql", func(c *gin.Context) {
		h.ContextHandler(c.Request.Context(), c.Writer, c.Request)
	})
}

func buildSchema() (graphql.Schema, error) {
	paymentType := buildPaymentType()
	walletType := buildWalletType()
	userType := buildUserType()
	merchantType := buildMerchantType()
	paymentSummaryType := buildPaymentSummaryType()

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"payment": &graphql.Field{
				Type: paymentType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolvePayment,
			},
			"payments": &graphql.Field{
				Type: graphql.NewList(paymentType),
				Args: graphql.FieldConfigArgument{
					"userId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"status": &graphql.ArgumentConfig{Type: graphql.String},
					"limit":  &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: resolvePayments,
			},
			"searchPayments": &graphql.Field{
				Type: graphql.NewList(paymentType),
				Args: graphql.FieldConfigArgument{
					"minAmount": &graphql.ArgumentConfig{Type: graphql.Float},
					"maxAmount": &graphql.ArgumentConfig{Type: graphql.Float},
					"currency":  &graphql.ArgumentConfig{Type: graphql.String},
					"status":    &graphql.ArgumentConfig{Type: graphql.String},
					"page":      &graphql.ArgumentConfig{Type: graphql.Int},
					"size":      &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: resolveSearchPayments,
			},
			"paymentSummary": &graphql.Field{
				Type: paymentSummaryType,
				Args: graphql.FieldConfigArgument{
					"startDate": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"endDate":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolvePaymentSummary,
			},
			"user": &graphql.Field{
				Type: userType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveUser,
			},
			"users": &graphql.Field{
				Type:    graphql.NewList(userType),
				Resolve: resolveUsers,
			},
			"merchant": &graphql.Field{
				Type: merchantType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveMerchant,
			},
			"merchants": &graphql.Field{
				Type:    graphql.NewList(merchantType),
				Resolve: resolveMerchants,
			},
			"wallet": &graphql.Field{
				Type: walletType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveWallet,
			},
			"wallets": &graphql.Field{
				Type:    graphql.NewList(walletType),
				Resolve: resolveWallets,
			},
			"walletByUserId": &graphql.Field{
				Type: walletType,
				Args: graphql.FieldConfigArgument{
					"userId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveWalletByUserID,
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"processPayment": &graphql.Field{
				Type: paymentType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(processPaymentInputType)},
				},
				Resolve: resolveProcessPayment,
			},
			"walletTransfer": &graphql.Field{
				Type: paymentType,
				Args: graphql.FieldConfigArgument{
					"walletId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"merchantId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"amount":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Float)},
				},
				Resolve: resolveWalletTransfer,
			},
			"topUpWallet": &graphql.Field{
				Type: walletType,
				Args: graphql.FieldConfigArgument{
					"walletId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"amount":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Float)},
				},
				Resolve: resolveTopUpWallet,
			},
			"processBatchPayments": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"payments": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.NewList(processPaymentInputType))},
				},
				Resolve: resolveProcessBatchPayments,
			},
			"refundPayment": &graphql.Field{
				Type: paymentType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveRefundPayment,
			},
			"createUser": &graphql.Field{
				Type: userType,
				Args: graphql.FieldConfigArgument{
					"email":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"fullName": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"status":   &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: resolveCreateUser,
			},
			"deleteUser": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveDeleteUser,
			},
			"createMerchant": &graphql.Field{
				Type: merchantType,
				Args: graphql.FieldConfigArgument{
					"name":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"apiKey": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveCreateMerchant,
			},
			"deleteMerchant": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolveDeleteMerchant,
			},
			"createWallet": &graphql.Field{
				Type: walletType,
				Args: graphql.FieldConfigArgument{
					"userId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"balance":  &graphql.ArgumentConfig{Type: graphql.Float},
					"currency": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: resolveCreateWallet,
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}
