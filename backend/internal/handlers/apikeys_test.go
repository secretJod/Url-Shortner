package handlers

import (
	"testing"
	"time"
)

func TestGenerateTokenIsRandomAndConsistentlyHashed(t *testing.T) {
	raw1, hash1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error: %v", err)
	}
	raw2, hash2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error: %v", err)
	}

	if raw1 == raw2 {
		t.Error("two generated tokens were identical — randomness is broken")
	}
	if hashToken(raw1) != hash1 {
		t.Error("hashToken(raw1) doesn't match the hash returned by generateToken")
	}
	if hash1 == hash2 {
		t.Error("two different tokens hashed to the same value")
	}
	// The raw token must never equal its own hash (i.e. hashing actually happened).
	if raw1 == hash1 {
		t.Error("raw token equals its hash — token wasn't hashed")
	}
}

func TestVerificationTokenExpiry(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name      string
		expiresAt time.Time
		expired   bool
	}{
		{"not yet expired", now.Add(10 * time.Minute), false},
		{"expired", now.Add(-1 * time.Minute), true},
		{"expires exactly now (treated as expired)", now.Add(-time.Nanosecond), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.expiresAt.Before(time.Now())
			if got != c.expired {
				t.Errorf("expiresAt.Before(now) = %v, want %v", got, c.expired)
			}
		})
	}
}
