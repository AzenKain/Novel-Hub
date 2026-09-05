package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	Period     = 30 * time.Second
	Digits     = 6
	secretSize = 20
	skew       = 1
)

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

var ErrInvalidSecret = errors.New("invalid TOTP secret")

func GenerateSecret() (string, error) {
	buf := make([]byte, secretSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return encoding.EncodeToString(buf), nil
}

func codeAt(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", Digits, value%1000000)
}

func Generate(secret string, now time.Time) (string, error) {
	key, err := encoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) == 0 {
		return "", ErrInvalidSecret
	}
	return codeAt(key, uint64(now.Unix())/uint64(Period/time.Second)), nil
}

// Returns the matched step counter so the caller can reject a replay: a code stays valid for the whole window, which is long enough for a shoulder-surfed code to be used twice.
func ValidateWithCounter(secret, code string, now time.Time) (uint64, bool) {
	key, err := encoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) == 0 {
		return 0, false
	}
	code = strings.TrimSpace(code)
	if len(code) != Digits {
		return 0, false
	}
	counter := uint64(now.Unix()) / uint64(Period/time.Second)
	for delta := -skew; delta <= skew; delta++ {
		candidate := counter
		switch {
		case delta < 0:
			step := uint64(-delta)
			if candidate < step {
				continue
			}
			candidate -= step
		case delta > 0:
			candidate += uint64(delta)
		}
		if subtle.ConstantTimeCompare([]byte(codeAt(key, candidate)), []byte(code)) == 1 {
			return candidate, true
		}
	}
	return 0, false
}

func Validate(secret, code string, now time.Time) bool {
	_, ok := ValidateWithCounter(secret, code, now)
	return ok
}

func ProvisioningURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", fmt.Sprintf("%d", Digits))
	query.Set("period", fmt.Sprintf("%d", int(Period/time.Second)))
	return "otpauth://totp/" + label + "?" + query.Encode()
}
