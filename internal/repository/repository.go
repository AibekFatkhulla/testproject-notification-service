package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

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

type EmailRepository interface {
	SaveLog(ctx context.Context, log EmailLog) error
}

type PostgresEmailRepository struct {
	db *sql.DB
}

func NewPostgresEmailRepository(db *sql.DB) *PostgresEmailRepository {
	return &PostgresEmailRepository{db: db}
}

func (r *PostgresEmailRepository) SaveLog(ctx context.Context, l EmailLog) error {
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
