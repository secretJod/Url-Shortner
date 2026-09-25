// Package mail sends transactional email (currently just verification
// links) via SMTP. Two modes are supported, selected at runtime by whether
// credentials are configured:
//
//   - No SMTPUser/SMTPPassword (the MailHog/dev default): plaintext,
//     unauthenticated, no TLS. MailHog and most local dev SMTP catchers
//     don't support STARTTLS or AUTH, so this path is kept working
//     unconditionally for `docker compose up` dev.
//   - SMTPUser/SMTPPassword present: the connection is upgraded with
//     STARTTLS and authenticated with AUTH PLAIN before sending. This is
//     required by every genuinely-free public SMTP relay (Gmail app
//     passwords, Brevo, SendGrid, ...), none of which accept unauthenticated
//     or unencrypted mail.
package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

// Mailer sends a single email. Kept as an interface so handlers can be
// tested with a fake implementation instead of hitting real SMTP.
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer sends mail via net/smtp. Best-effort: callers should not fail a
// request purely because a verification email couldn't be sent, though the
// key-issuance flow does surface a 502 so the user knows to retry rather
// than wait forever for an email that never arrives.
type SMTPMailer struct {
	Host     string
	Port     string
	From     string
	User     string
	Password string
}

// NewSMTPMailer builds a Mailer targeting host:port with the given From
// address. If user/password are non-empty, Send authenticates the
// connection with STARTTLS + AUTH PLAIN; otherwise it falls back to the
// plaintext, unauthenticated MailHog-compatible path.
func NewSMTPMailer(host, port, from, user, password string) *SMTPMailer {
	return &SMTPMailer{Host: host, Port: port, From: from, User: user, Password: password}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n",
		m.From, to, subject, body)

	if m.User == "" && m.Password == "" {
		// No auth — MailHog (and most local dev SMTP catchers) don't require it.
		return smtp.SendMail(addr, nil, m.From, []string{to}, []byte(msg))
	}

	return m.sendAuthenticated(addr, to, []byte(msg))
}

// sendAuthenticated sends mail over a connection upgraded with STARTTLS and
// authenticated with AUTH PLAIN. smtp.PlainAuth refuses to hand over
// credentials on a connection it doesn't believe is encrypted, so STARTTLS
// must succeed before Auth is called — this function enforces that
// ordering explicitly rather than relying on smtp.SendMail's own (looser)
// behavior.
//
// If the server does not advertise STARTTLS support, this fails closed
// rather than falling back to sending AUTH PLAIN in the clear: PLAIN auth
// sends the password essentially as base64-encoded plaintext, so silently
// downgrading would leak real mailbox credentials to anything on the network
// path. An operator who configured SMTP_USER/SMTP_PASSWORD wants
// authenticated mail; if the relay can't do that securely, surfacing a clear
// error is better than a silent credential leak.
func (m *SMTPMailer) sendAuthenticated(addr, to string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", addr, err)
	}
	defer client.Close()

	ok, _ := client.Extension("STARTTLS")
	if !ok {
		return fmt.Errorf("mail: SMTP_USER/SMTP_PASSWORD configured but server at %s does not advertise STARTTLS; refusing to send credentials over an unencrypted connection", addr)
	}

	tlsConfig := &tls.Config{ServerName: m.Host}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("mail: STARTTLS with %s: %w", addr, err)
	}

	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mail: authenticating to %s: %w", addr, err)
	}

	if err := client.Mail(m.From); err != nil {
		return fmt.Errorf("mail: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail: RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("mail: writing message body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: closing message body: %w", err)
	}

	return client.Quit()
}
