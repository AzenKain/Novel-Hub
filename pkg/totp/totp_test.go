package totp

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 Appendix B, SHA-1 rows. The published vectors use the ASCII secret
// "12345678901234567890"; base32 of that is GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ.
func TestValidateMatchesRFC6238Vectors(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	vectors := []struct {
		unix int64
		code string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}

	for _, v := range vectors {
		if !Validate(secret, v.code, time.Unix(v.unix, 0)) {
			t.Errorf("RFC 6238 vector at t=%d: %s rejected", v.unix, v.code)
		}
	}
}

func TestValidateRejectsWrongAndMalformedCodes(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	at := time.Unix(59, 0)

	for _, code := range []string{"000000", "28708", "2870823", "", "abcdef", "287083"} {
		if Validate(secret, code, at) {
			t.Errorf("%q was accepted", code)
		}
	}
	if Validate("not-base32!!", "287082", at) {
		t.Error("a malformed secret was accepted")
	}
	if Validate("", "287082", at) {
		t.Error("an empty secret was accepted")
	}
}

// A phone clock is never exactly right; one step either side must still work, two must not.
func TestValidateToleratesOneStepOfDrift(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	base := time.Unix(1111111109, 0)
	const code = "081804"

	if !Validate(secret, code, base.Add(-Period)) {
		t.Error("one step behind was rejected")
	}
	if !Validate(secret, code, base.Add(Period)) {
		t.Error("one step ahead was rejected")
	}
	if Validate(secret, code, base.Add(-3*Period)) {
		t.Error("three steps behind was accepted; the replay window is too wide")
	}
	if Validate(secret, code, base.Add(3*Period)) {
		t.Error("three steps ahead was accepted; the replay window is too wide")
	}
}

func TestGenerateSecretIsUsableAndUnique(t *testing.T) {
	seen := make(map[string]bool, 32)
	for range 32 {
		secret, err := GenerateSecret()
		if err != nil {
			t.Fatalf("GenerateSecret: %v", err)
		}
		if seen[secret] {
			t.Fatal("GenerateSecret returned a duplicate")
		}
		seen[secret] = true

		now := time.Now()
		key, err := encoding.DecodeString(secret)
		if err != nil {
			t.Fatalf("generated secret is not decodable: %v", err)
		}
		code := codeAt(key, uint64(now.Unix())/uint64(Period/time.Second))
		if !Validate(secret, code, now) {
			t.Fatal("a freshly generated secret did not validate its own code")
		}
	}
}

func TestProvisioningURICarriesTheSecretAndIssuer(t *testing.T) {
	uri := ProvisioningURI("NovelHub", "reader@example.com", "GEZDGNBVGY3TQOJQ")
	for _, want := range []string{
		"otpauth://totp/",
		"secret=GEZDGNBVGY3TQOJQ",
		"issuer=NovelHub",
		"digits=6",
		"period=30",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("provisioning URI missing %q: %s", want, uri)
		}
	}
}
