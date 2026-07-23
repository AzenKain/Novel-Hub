package crypto_test

import (
	"testing"

	"novelhub/pkg/crypto"
)

func TestAESEncryptionDecryption(t *testing.T) {
	t.Setenv("DB_ENCRYPTION_KEY", "test-encryption-key")
	original := "secret_oauth_token_123456"

	encrypted, err := crypto.EncryptAES(original)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if encrypted == original {
		t.Fatalf("expected encrypted text to differ from original")
	}

	decrypted, err := crypto.DecryptAES(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != original {
		t.Fatalf("expected decrypted '%s', got '%s'", original, decrypted)
	}
}
