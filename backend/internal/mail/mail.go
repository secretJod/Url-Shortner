// Package mail sends transactional email (currently just verification
// links) via plain SMTP. Designed against a local MailHog instance for
// dev — no auth, no TLS — since the project has a zero-cost requirement
// and MailHog is the free local substitute for a real mail provider.
package mail

import (
	"fmt"
	"net/smtp"
)

// Mailer sends a single email. Kept as an interface so handlers can be
// tested with a fake implementation instead of hitting real SMTP.
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer sends mail via net/smtp to a plaintext, unauthenticated SMTP
// server (MailHog). Best-effort: callers should not fail a request purely
// because a verification email couldn't be sent, though the key-issuance
// flow does surface a 502 so the user knows to retry rather than wait
// forever for an email that never arrives.
type SMTPMailer struct {
	Host string
	Port string
	From string
}

// NewSMTPMailer builds a Mailer targeting host:port with the given From address.
func NewSMTPMailer(host, port, from string) *SMTPMailer {
	return &SMTPMailer{Host: host, Port: port, From: from}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n",
		m.From, to, subject, body)

	// No auth — MailHog (and most local dev SMTP catchers) don't require it.
	return smtp.SendMail(addr, nil, m.From, []string{to}, []byte(msg))
}
