package service

import (
	"context"
	"database/sql"
	"fmt"
	"notification-service/internal/domain"
	"notification-service/internal/repository"
	"notification-service/internal/sender"
	"notification-service/internal/validator"
	"time"

	log "github.com/sirupsen/logrus"
)

type NotificationServiceInterface interface {
	ProcessPurchase(ctx context.Context, purchase domain.PurchaseInfo) error
}

type NotificationService struct {
	emailSender     sender.EmailSender
	emailRepository repository.EmailRepository
}

func NewNotificationService(emailSender sender.EmailSender, emailRepository repository.EmailRepository) *NotificationService {
	return &NotificationService{emailSender: emailSender, emailRepository: emailRepository}
}

func (s *NotificationService) ProcessPurchase(ctx context.Context, purchase domain.PurchaseInfo) error {
	if err := validator.ValidatePurchaseInfo(purchase); err != nil {
		log.WithFields(log.Fields{
			"error":          err,
			"purchase":       purchase,
			"transaction_id": purchase.TransactionID,
		}).Error("Purchase info validation failed")
		return fmt.Errorf("validation error: %w", err)
	}

	subject := "Покупка монет успешно завершена!"
	body := fmt.Sprintf(
		"Здравствуйте!\n\nВы успешно приобрели %d монет.\nID вашей транзакции: %s\n\nСпасибо за покупку!",
		purchase.CoinsPurchased,
		purchase.TransactionID,
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Retry sending email up to 3 times with exponential backoff
	maxAttempts := 3
	initialDelay := 1 * time.Second
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = s.emailSender.SendEmail(ctx, purchase.UserEmail, subject, body)
		if err == nil {
			if attempt > 1 {
				log.WithFields(log.Fields{
					"attempt":      attempt,
					"max_attempts": maxAttempts,
					"email":        purchase.UserEmail,
				}).Info("Email sent successfully after retry")
			}
			break
		}

		// If error and not last attempt - retry
		if attempt < maxAttempts {
			log.WithFields(log.Fields{
				"attempt":      attempt,
				"max_attempts": maxAttempts,
				"error":        err,
				"email":        purchase.UserEmail,
			}).Warn("Failed to send email, retrying...")

			time.Sleep(initialDelay)
			initialDelay *= 2
		}
	}

	logEntry := repository.EmailLog{
		TransactionID:  purchase.TransactionID,
		RecipientEmail: purchase.UserEmail,
		Subject:        subject,
	}

	if err != nil {
		log.WithError(err).Error("Failed to send confirmation email via SMTP")
		logEntry.Status = repository.StatusFailed
		logEntry.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
	} else {
		log.WithField("email", purchase.UserEmail).Info("Confirmation email sent successfully via SMTP")
		logEntry.Status = repository.StatusSent
	}

	if err := s.emailRepository.SaveLog(ctx, logEntry); err != nil {
		log.WithError(err).Error("Failed to save email log to database")
		return err
	}

	return nil
}
