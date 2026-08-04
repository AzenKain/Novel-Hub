package kobo

import (
	"crypto/rand"
	"encoding/base64"
)

// RandomToken mints one of the throwaway tokens /v1/auth/device hands the device:
// base64(urandom(24)), same as calibre-web. The value is never stored or checked — the real
// credential is the token in the URL path — but it is generated with crypto/rand anyway,
// because a predictable "token" in an auth response is the kind of thing that stops being
// harmless the moment someone starts trusting it.
func RandomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
