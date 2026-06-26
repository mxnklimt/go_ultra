package session

import (
	"testing"
	"time"
)

func TestNewToken_Length(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken returned error: %v", err)
	}
	// 32 bytes raw -> base64.RawURLEncoding 无填充，长度 = ceil(32*8/6) = 43
	if len(tok) != 43 {
		t.Fatalf("token length = %d, want 43; token=%q", len(tok), tok)
	}
}

func TestNewToken_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		tok, err := NewToken()
		if err != nil {
			t.Fatalf("NewToken returned error at i=%d: %v", i, err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token generated at i=%d: %q", i, tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestNewToken_URLSafe(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken returned error: %v", err)
	}
	for _, r := range tok {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			t.Fatalf("token contains non-url-safe char %q in %q", r, tok)
		}
	}
}

func TestConstants(t *testing.T) {
	if PlayerSessionTTL != 30*24*time.Hour {
		t.Fatalf("PlayerSessionTTL = %v, want 720h", PlayerSessionTTL)
	}
	if AdminSessionTTL != 30*time.Minute {
		t.Fatalf("AdminSessionTTL = %v, want 30m", AdminSessionTTL)
	}
	if PlayerCookieName != "go_ultra_session" {
		t.Fatalf("PlayerCookieName = %q, want go_ultra_session", PlayerCookieName)
	}
	if AdminCookieName != "go_ultra_admin" {
		t.Fatalf("AdminCookieName = %q, want go_ultra_admin", AdminCookieName)
	}
}
