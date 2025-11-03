CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS email_logs (
                                          id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id TEXT,
    recipient_email TEXT NOT NULL,
    subject TEXT,
    status TEXT NOT NULL,
    error_message TEXT,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_email_logs_transaction_id ON email_logs (transaction_id);