package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidWebhookSignature(t *testing.T) {
	body := []byte(`{"status":"PAID"}`)
	secret := "test-provider-secret-with-more-than-32-bytes"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	if !validWebhookSignature(body, signature, secret) {
		t.Fatal("expected matching HMAC signature to pass")
	}
	if validWebhookSignature([]byte(`{"status":"FAILED"}`), signature, secret) {
		t.Fatal("expected modified webhook body to fail signature validation")
	}
	if validWebhookSignature(body, "invalid", secret) {
		t.Fatal("expected malformed signature to fail")
	}
}
