package sender

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type SMTPEmailSender struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPEmailSender(host, port, user, pass, from string) *SMTPEmailSender {
	return &SMTPEmailSender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTPEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	e := email.NewEmail()
	e.From = s.from
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte(body)

	return e.Send(addr, auth)
}
