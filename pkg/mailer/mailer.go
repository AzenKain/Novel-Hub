package mailer

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	Path     string
	Data     []byte
}

type Mailer interface {
	SendEmail(to string, subject string, body string, attachment *Attachment) error
}

type smtpMailer struct{ config SMTPConfig }

func NewSMTPMailer(config SMTPConfig) Mailer { return &smtpMailer{config: config} }

func rejectHeaderInjection(values ...string) error {
	for _, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("email header contains a newline")
		}
	}
	return nil
}

func (m *smtpMailer) SendEmail(to, subject, body string, attachment *Attachment) error {
	if m.config.Host == "" || m.config.Port == 0 {
		return fmt.Errorf("SMTP configuration incomplete")
	}
	if err := rejectHeaderInjection(to, subject, m.config.FromEmail); err != nil {
		return err
	}
	ips, err := net.LookupIP(m.config.Host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("failed to resolve SMTP host: %w", err)
	}
	for _, ip := range ips {
		if netx.IsPrivateIP(ip) {
			return fmt.Errorf("connecting to private IP address is forbidden for SMTP")
		}
	}
	addr := net.JoinHostPort(ips[0].String(), strconv.Itoa(m.config.Port))
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("SMTP STARTTLS failed: %w", err)
		}
	}
	if m.config.Username != "" && m.config.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)); err != nil {
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
	if err := writeMessage(w, m.config.FromEmail, to, subject, body, attachment); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func writeMessage(w io.Writer, from, to, subject, body string, attachment *Attachment) error {
	if _, err := fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n", from, to, subject); err != nil {
		return err
	}
	if attachment == nil {
		_, err := fmt.Fprintf(w, "Content-Type: text/plain; charset=utf-8\r\n\r\n%s", body)
		return err
	}
	name := filepath.Base(attachment.Filename)
	if err := rejectHeaderInjection(name); err != nil {
		return err
	}
	const boundary = "NOVELHUB_MAIL_BOUNDARY"
	if _, err := fmt.Fprintf(w, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n--%s\r\nContent-Type: application/octet-stream; name=\"%s\"\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=\"%s\"\r\n\r\n", boundary, boundary, body, boundary, name, name); err != nil {
		return err
	}
	var src io.ReadCloser
	if attachment.Path != "" {
		file, err := os.Open(attachment.Path)
		if err != nil {
			return err
		}
		src = file
	} else {
		src = io.NopCloser(strings.NewReader(string(attachment.Data)))
	}
	defer src.Close()
	lineWriter := &base64LineWriter{w: w}
	encoder := base64.NewEncoder(base64.StdEncoding, lineWriter)
	if _, err := io.Copy(encoder, src); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := lineWriter.flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\r\n--%s--\r\n", boundary)
	return err
}

type base64LineWriter struct {
	w   io.Writer
	buf []byte
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	original := len(p)
	w.buf = append(w.buf, p...)
	for len(w.buf) >= 76 {
		if _, err := w.w.Write(append(append([]byte(nil), w.buf[:76]...), '\r', '\n')); err != nil {
			return 0, err
		}
		w.buf = w.buf[76:]
	}
	return original, nil
}

func (w *base64LineWriter) flush() error {
	if len(w.buf) == 0 {
		return nil
	}
	_, err := w.w.Write(w.buf)
	w.buf = nil
	return err
}
