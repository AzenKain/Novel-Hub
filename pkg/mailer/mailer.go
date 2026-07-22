package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"path/filepath"
	"strconv"
	"time"

	"novelhub/pkg/netx"
)

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
}

type Attachment struct {
	Filename string
	Data     []byte
}

type Mailer interface {
	SendEmail(to string, subject string, body string, attachment *Attachment) error
}

type smtpMailer struct {
	config SMTPConfig
}

func NewSMTPMailer(config SMTPConfig) Mailer {
	return &smtpMailer{config: config}
}

func (m *smtpMailer) SendEmail(to string, subject string, body string, attachment *Attachment) error {
	if m.config.Host == "" || m.config.Port == 0 {
		return fmt.Errorf("SMTP configuration incomplete")
	}

	ips, err := net.LookupIP(m.config.Host)
	if err != nil {
		return fmt.Errorf("failed to resolve SMTP host: %w", err)
	}
	for _, ip := range ips {
		if netx.IsPrivateIP(ip) {
			return fmt.Errorf("connecting to private IP address is forbidden for SMTP")
		}
	}

	addr := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))

	var auth smtp.Auth
	if m.config.Username != "" && m.config.Password != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	var msg bytes.Buffer
	boundary := "---NOVELHUB_MAIL_BOUNDARY---"

	msg.WriteString(fmt.Sprintf("From: %s\r\n", m.config.FromEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if attachment != nil {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body)
		msg.WriteString("\r\n\r\n")

		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", filepath.Base(attachment.Filename)))
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filepath.Base(attachment.Filename)))

		encoded := base64.StdEncoding.EncodeToString(attachment.Data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			msg.WriteString(encoded[i:end])
			msg.WriteString("\r\n")
		}
		msg.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body)
	}

	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(m.config.FromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg.Bytes())
	if err != nil {
		return err
	}
	return w.Close()
}
