package domain

import "database/sql"

type PurchaseInfo struct {
	TransactionID  string `json:"transaction_id"`
	UserID         string `json:"user_id"`
	UserEmail      string `json:"user_email"`
	CoinsPurchased int    `json:"coins_purchased"`
	Provider       string `json:"provider"`
	Country        string `json:"country"`
	Funnel         string `json:"funnel"`
	ProductID      string `json:"product_id"`
}

type RefundInfo struct {
	RefundID      string `json:"refund_id"`
	TransactionID string `json:"transaction_id"`
	UserID        string `json:"user_id"`
	UserEmail     string `json:"user_email"`
	Amount        int64  `json:"amount"`         // in cents
	CoinsDeducted int64  `json:"coins_deducted"` // coins that were deducted
	Reason        string `json:"reason,omitempty"`
	ProcessedAt   string `json:"processed_at"` // ISO 8601 timestamp
}

type EmailStatus string

const (
	StatusSent   EmailStatus = "sent"
	StatusFailed EmailStatus = "failed"
)

type EmailLog struct {
	TransactionID  string
	RecipientEmail string
	Subject        string
	Status         EmailStatus
	ErrorMessage   sql.NullString
}
