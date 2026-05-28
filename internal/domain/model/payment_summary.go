package model

type PaymentSummary struct {
	TotalsByStatus map[string]float64 `json:"totalsByStatus"`
	TotalCount     int                `json:"totalCount"`
	TotalAmount    float64            `json:"totalAmount"`
}

func NewPaymentSummary() *PaymentSummary {
	return &PaymentSummary{
		TotalsByStatus: make(map[string]float64),
	}
}
