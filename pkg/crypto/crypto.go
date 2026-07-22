package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"sync"

	"novelhub/pkg/config"
)

var (
	keyCache []byte
	keyOnce  sync.Once
)

func getEncryptionKey() []byte {
	keyOnce.Do(func() {
		secret := config.GetConfigWithDefault("DB_ENCRYPTION_KEY", "")
		if secret == "" {
			secret = config.GetConfigWithDefault("JWT_SECRET", "novelhub-default-encryption-secret-32b")
		}
		hash := sha256.Sum256([]byte(secret))
		keyCache = hash[:]
	})
	return keyCache
}

// EncryptAES encrypts plain text using AES-256-GCM and returns a base64 encoded string.
func EncryptAES(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES decrypts a base64 encoded AES-256-GCM string.
func DecryptAES(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	key := getEncryptionKey()
	data, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext length")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
