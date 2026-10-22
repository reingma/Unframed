package api

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseToken(t *testing.T) {
	if tok, err := ParseToken(""); err != nil || tok.enabled() {
		t.Fatalf("empty token = %v, %v; want the disabled zero Token", tok, err)
	}
	for range 100 {
		key := make([]byte, 32)
		rand.Read(key)
		value := base64.RawURLEncoding.EncodeToString(key)
		tok, err := ParseToken(value)
		if err != nil || !tok.matches(value) {
			t.Fatalf("generated base64url token %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{
		strings.Repeat("a", MinTokenLength-1),
		strings.Repeat("a", MaxTokenLength+1),
		strings.Repeat("a", 31) + "+",
		strings.Repeat("a", 31) + "/",
		strings.Repeat("a", 31) + "=",
		strings.Repeat("a", 31) + "%",
		strings.Repeat("a", 31) + " ",
		strings.Repeat("a", 30) + "é",
	} {
		if _, err := ParseToken(value); err == nil {
			t.Errorf("ParseToken(%q) accepted", value)
		}
	}
}
