package domain

type PurchaseInfo struct {
	TransactionID string `json:"transaction_id"`
	UserID        string `json:"user_id"`
	UserEmail     string `json:"user_email"`
	CoinsPurchased int    `json:"coins_purchased"`
	Provider       string `json:"provider"`
	Country        string `json:"country"`
	Funnel         string `json:"funnel"`
}