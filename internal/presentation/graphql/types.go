package graphql

import (
	"github.com/graphql-go/graphql"
)

var processPaymentInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "ProcessPaymentInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"userId":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"merchantId": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"amount":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Float)},
		"type":       &graphql.InputObjectFieldConfig{Type: graphql.String},
	},
})

func buildPaymentType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Payment",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"userId":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"merchantId": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"walletId":   &graphql.Field{Type: graphql.String},
			"amount":     &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"type":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"metadata":   &graphql.Field{Type: graphql.String},
			"createdAt":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})
}

func buildWalletType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Wallet",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"userId":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"balance":   &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"currency":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})
}

func buildUserType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"email":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"fullName":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})
}

func buildMerchantType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Merchant",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"name":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"apiKey":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})
}

func buildPaymentSummaryType() *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "PaymentSummary",
		Fields: graphql.Fields{
			"totalsByStatus": &graphql.Field{
				Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
					Name: "StatusTotal",
					Fields: graphql.Fields{
						"status": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
						"total":  &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
					},
				})),
			},
			"totalCount":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"totalAmount": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
		},
	})
}
