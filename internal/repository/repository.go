package repository

import (
	"context"
	"database/sql"
	"fmt"
	"notification-service/internal/domain"
	"time"

	log "github.com/sirupsen/logrus"
)

type postgresEmailRepository struct {
	db *sql.DB
}

func NewPostgresEmailRepository(db *sql.DB) *postgresEmailRepository {
	return &postgresEmailRepository{db: db}
}

func (r *postgresEmailRepository) SaveLog(ctx context.Context, l domain.EmailLog) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"transaction_id": l.TransactionID,
		"recipient_email": l.RecipientEmail,
		"subject": l.Subject,
		"status": l.Status,
		"error_message": l.ErrorMessage,
	}).Info("Saving email log to database")

	const query = `
        INSERT INTO email_logs (transaction_id, recipient_email, subject, status, error_message)
        VALUES ($1, $2, $3, $4, $5);
    `

	if _, err := r.db.ExecContext(ctx, query, l.TransactionID, l.RecipientEmail, l.Subject, string(l.Status), nullStringOrNil(l.ErrorMessage)); err != nil {
		return fmt.Errorf("failed to insert email log: %w", err)
	}
	return nil
}

func nullStringOrNil(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}
