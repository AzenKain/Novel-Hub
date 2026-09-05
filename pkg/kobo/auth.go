package kobo

import (
	"crypto/rand"
	"encoding/base64"
)

// RandomToken mints one of the throwaway tokens /v1/auth/device hands the device: base64(urandom(24)), same as calibre-web.
func RandomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
