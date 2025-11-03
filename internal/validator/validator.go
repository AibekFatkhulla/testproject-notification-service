package validator

import (
	"errors"
	"regexp"
	"strings"
	"notification-service/internal/domain"
)

var (
	ErrEmptyEmail           = errors.New("email is empty")
	ErrInvalidEmailFormat   = errors.New("invalid email format")
	ErrEmptyTransactionID   = errors.New("transaction ID is empty")
	ErrEmptyUserID          = errors.New("user ID is empty")
	ErrInvalidUserID        = errors.New("user ID is invalid")
	ErrInvalidCoins         = errors.New("coins purchased must be greater than 0")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return ErrEmptyEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmailFormat
	}
	return nil
}

func ValidateTransactionID(transactionID string) error {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return ErrEmptyTransactionID
	}
	return nil
}

func ValidateUserID(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrEmptyUserID
	}
	if len(userID) < 3 {
		return ErrInvalidUserID
	}
	return nil
}

func ValidateCoins(coins int) error {
	if coins <= 0 {
		return ErrInvalidCoins
	}
	return nil
}

func ValidatePurchaseInfo(purchaseInfo domain.PurchaseInfo) error {
	if err := ValidateEmail(purchaseInfo.UserEmail); err != nil {
		return err
	}
	if err := ValidateTransactionID(purchaseInfo.TransactionID); err != nil {
		return err
	}
	if err := ValidateUserID(purchaseInfo.UserID); err != nil {
		return err
	}
	if err := ValidateCoins(purchaseInfo.CoinsPurchased); err != nil {
		return err
	}
	return nil
}
