package mail

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// fakeSMTPServer is a minimal, single-connection SMTP server good enough to
// exercise SMTPMailer's two send paths without a real relay. It is not a
// substitute for testing against an actual SMTP provider (see test comments
// below for what remains unverified).
type fakeSMTPServer struct {
	ln net.Listener
	// advertiseSTARTTLS controls whether EHLO's response lists the STARTTLS
	// extension. When false, SMTPMailer.sendAuthenticated must refuse to
	// proceed rather than send credentials in the clear.
	advertiseSTARTTLS bool
}

func newFakeSMTPServer(t *testing.T, advertiseSTARTTLS bool) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSMTPServer{ln: ln, advertiseSTARTTLS: advertiseSTARTTLS}
	go s.serveOne(t)
	return s
}

func (s *fakeSMTPServer) addr() (host, port string) {
	host, port, _ = net.SplitHostPort(s.ln.Addr().String())
	return host, port
}

// serveOne handles exactly one connection with just enough of the SMTP
// protocol to let net/smtp's client complete a full plaintext send. It does
// not implement STARTTLS itself (a real TLS handshake is out of scope for a
// unit test); it only ever advertises or omits the extension so we can
// assert on SMTPMailer's client-side reaction to each case.
func (s *fakeSMTPServer) serveOne(t *testing.T) {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	writeLine := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	writeLine("220 fake.smtp.test ESMTP")

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.Fields(line)[0])

		switch cmd {
		case "EHLO", "HELO":
			if s.advertiseSTARTTLS {
				writeLine("250-fake.smtp.test greets you")
				writeLine("250 STARTTLS")
			} else {
				writeLine("250 fake.smtp.test greets you")
			}
		case "MAIL":
			writeLine("250 OK")
		case "RCPT":
			writeLine("250 OK")
		case "DATA":
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
			}
			writeLine("250 OK: queued")
		case "QUIT":
			writeLine("221 Bye")
			return
		default:
			writeLine("500 unrecognized command")
		}
	}
}

func TestSMTPMailer_Send_NoCredentials_PlaintextPath(t *testing.T) {
	srv := newFakeSMTPServer(t, false)
	host, port := srv.addr()

	m := NewSMTPMailer(host, port, "no-reply@example.com", "", "")
	if err := m.Send("user@example.com", "subject", "body"); err != nil {
		t.Fatalf("Send (unauthenticated path) failed: %v", err)
	}
}

func TestSMTPMailer_Send_CredentialsConfigured_RequiresSTARTTLS(t *testing.T) {
	// Server does NOT advertise STARTTLS. With credentials configured, Send
	// must fail closed rather than send AUTH PLAIN in the clear.
	srv := newFakeSMTPServer(t, false)
	host, port := srv.addr()

	m := NewSMTPMailer(host, port, "no-reply@example.com", "smtpuser", "smtppass")
	err := m.Send("user@example.com", "subject", "body")
	if err == nil {
		t.Fatal("expected Send to fail when server does not advertise STARTTLS, got nil error")
	}
	if !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected error to mention STARTTLS, got: %v", err)
	}
}

// TestSMTPMailer_Send_CredentialsConfigured_STARTTLSAdvertisedButHandshakeFails
// documents the boundary of what this fake server can exercise: it
// advertises STARTTLS but cannot perform a real TLS handshake, so the
// client-side StartTLS call is expected to fail. This at least proves
// SMTPMailer reaches the STARTTLS step (rather than skipping it) once the
// server offers the extension. The full authenticated happy path — STARTTLS
// handshake succeeding, then AUTH PLAIN, then delivery — is NOT covered by
// any test here; see package-level notes in the review report for why.
func TestSMTPMailer_Send_CredentialsConfigured_STARTTLSAdvertisedButHandshakeFails(t *testing.T) {
	srv := newFakeSMTPServer(t, true)
	host, port := srv.addr()

	m := NewSMTPMailer(host, port, "no-reply@example.com", "smtpuser", "smtppass")
	err := m.Send("user@example.com", "subject", "body")
	if err == nil {
		t.Fatal("expected Send to fail (fake server can't complete a real TLS handshake), got nil error")
	}
	if !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected error to be about the STARTTLS handshake, got: %v", err)
	}
}

func TestNewSMTPMailer_FieldsSet(t *testing.T) {
	m := NewSMTPMailer("smtp.example.com", "587", "from@example.com", "user", "pass")
	if m.Host != "smtp.example.com" || m.Port != "587" || m.From != "from@example.com" ||
		m.User != "user" || m.Password != "pass" {
		t.Fatalf("NewSMTPMailer did not set fields correctly: %+v", m)
	}
}
