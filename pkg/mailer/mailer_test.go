package mailer

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPMailerValidation(t *testing.T) {
	mailer := NewSMTPMailer(SMTPConfig{
		Host: "",
		Port: 0,
	})
	err := mailer.SendEmail("test@example.com", "Subject", "Body", nil)
	if err == nil {
		t.Fatalf("expected error on incomplete SMTP config, got nil")
	}
}

func fakeSMTP(t *testing.T, tlsConfig *tls.Config, announceStartTLS bool) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
				reader := bufio.NewReader(conn)
				write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

				write("220 fake ESMTP")
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					command := strings.ToUpper(strings.TrimSpace(line))
					switch {
					case strings.HasPrefix(command, "EHLO"):
						if announceStartTLS {
							write("250-fake")
							write("250 STARTTLS")
						} else {
							write("250 fake")
						}
					case strings.HasPrefix(command, "QUIT"):
						write("221 bye")
						return
					default:
						write("250 ok")
					}
				}
			}()
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return number
}

func localhostTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
}

func TestTestConnectionBlocksPrivateHostUnlessAllowed(t *testing.T) {
	port := fakeSMTP(t, nil, false)
	config := SMTPConfig{Host: "localhost", Port: port, TLSMode: TLSModeNone}

	err := TestConnection(config)
	if err == nil || !strings.Contains(err.Error(), "private IP") {
		t.Fatalf("private host should be refused by default, got %v", err)
	}

	config.AllowPrivateNetworks = true
	if err := TestConnection(config); err != nil {
		t.Fatalf("private host should connect once allowed: %v", err)
	}
}

func TestTestConnectionRequiresStartTLSWhenSelected(t *testing.T) {
	port := fakeSMTP(t, nil, false)
	config := SMTPConfig{
		Host:                 "localhost",
		Port:                 port,
		TLSMode:              TLSModeStartTLS,
		AllowPrivateNetworks: true,
	}
	err := TestConnection(config)
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("starttls mode must not silently downgrade, got %v", err)
	}
}

func TestTestConnectionImplicitTLS(t *testing.T) {
	port := fakeSMTP(t, localhostTLSConfig(t), false)
	config := SMTPConfig{
		Host:                 "localhost",
		Port:                 port,
		TLSMode:              TLSModeImplicit,
		AllowPrivateNetworks: true,
	}
	if err := TestConnection(config); err != nil && !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("implicit TLS should reach the TLS handshake, got %v", err)
	}

	config.Port = fakeSMTP(t, nil, false)
	if err := TestConnection(config); err == nil {
		t.Fatal("implicit TLS against a plaintext server should fail")
	}
}
