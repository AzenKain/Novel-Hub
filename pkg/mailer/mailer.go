package mailer

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"novelhub/pkg/netx"
)

const (
	TLSModeNone     = "none"
	TLSModeStartTLS = "starttls"
	TLSModeImplicit = "implicit_tls"
)

type SMTPConfig struct {
	Host                 string
	Port                 int
	Username             string
	Password             string
	FromEmail            string
	TLSMode              string
	AllowPrivateNetworks bool
	MaxAttachmentMB      int
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

// Everything after "mailto:" lands in url.Opaque, including a ?subject= that is not a recipient.
func ParseRecipients(target string) ([]string, error) {
	trimmed := strings.TrimSpace(target)
	if !strings.HasPrefix(strings.ToLower(trimmed), "mailto:") {
		return nil, fmt.Errorf("email target must start with mailto:")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid mailto target: %w", err)
	}
	list := parsed.Opaque
	if list == "" {
		list = strings.TrimPrefix(trimmed, "mailto:")
	}
	if index := strings.IndexByte(list, '?'); index >= 0 {
		list = list[:index]
	}
	if decoded, err := url.QueryUnescape(list); err == nil {
		list = decoded
	}

	seen := make(map[string]bool)
	out := make([]string, 0, 4)
	for _, part := range strings.Split(list, ",") {
		address := strings.TrimSpace(part)
		if address == "" {
			continue
		}
		if err := rejectHeaderInjection(address); err != nil {
			return nil, err
		}
		parsedAddress, err := mail.ParseAddress(address)
		if err != nil {
			return nil, fmt.Errorf("invalid email recipient %q", address)
		}
		if seen[parsedAddress.Address] {
			continue
		}
		seen[parsedAddress.Address] = true
		out = append(out, parsedAddress.Address)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("email target has no recipients")
	}
	return out, nil
}

func (m *smtpMailer) SendEmail(to, subject, body string, attachment *Attachment) error {
	if err := rejectHeaderInjection(to, subject, m.config.FromEmail); err != nil {
		return err
	}
	client, err := connect(m.config)
	if err != nil {
		return err
	}
	defer client.Close()
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

func TestConnection(config SMTPConfig) error {
	client, err := connect(config)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Quit()
}

// Dials resolved IPs, not the hostname, so IsPrivateIP screens what is actually connected to.
func connect(config SMTPConfig) (*smtp.Client, error) {
	if config.Host == "" || config.Port == 0 {
		return nil, fmt.Errorf("SMTP configuration incomplete")
	}
	ips, err := net.LookupIP(config.Host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("failed to resolve SMTP host: %w", err)
	}
	if !config.AllowPrivateNetworks {
		for _, ip := range ips {
			if netx.IsPrivateIP(ip) {
				return nil, fmt.Errorf("connecting to private IP address is forbidden for SMTP")
			}
		}
	}
	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 15 * time.Second}

	var conn net.Conn
	for _, ip := range ips {
		addr := net.JoinHostPort(ip.String(), strconv.Itoa(config.Port))
		if config.TLSMode == TLSModeImplicit {
			conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		} else {
			conn, err = dialer.Dial("tcp", addr)
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if config.TLSMode != TLSModeImplicit && config.TLSMode != TLSModeNone {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP STARTTLS failed: %w", err)
		}
	}
	if config.Username != "" && config.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host)); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	return client, nil
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
