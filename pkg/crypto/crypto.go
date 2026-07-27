package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"novelhub/pkg/config"
)

const encryptedPrefix = "enc:v1:"

func getEncryptionKey() ([]byte, error) {
	secret, err := config.GetConfig("DB_ENCRYPTION_KEY")
	if err != nil {
		return nil, errors.New("DB_ENCRYPTION_KEY is required")
	}
	hash := sha256.Sum256([]byte(secret))
	return hash[:], nil
}

func EncryptAES(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := getEncryptionKey()
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

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAES(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(cryptoText, encryptedPrefix) {
		return "", errors.New("unencrypted secret rejected")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(cryptoText, encryptedPrefix))
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
