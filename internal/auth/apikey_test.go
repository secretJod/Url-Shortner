package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKeyIsRandomAndConsistentlyHashed(t *testing.T) {
	raw1, hash1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	raw2, hash2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	if raw1 == raw2 {
		t.Error("two generated keys were identical — randomness is broken")
	}
	if !strings.HasPrefix(raw1, keyPrefix) {
		t.Errorf("key %q missing expected prefix %q", raw1, keyPrefix)
	}
	if HashKey(raw1) != hash1 {
		t.Error("HashKey(raw1) doesn't match the hash returned by GenerateAPIKey")
	}
	if hash1 == hash2 {
		t.Error("two different keys hashed to the same value")
	}
}

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		header  string
		want    string
		wantErr bool
	}{
		{"Bearer usk_abc123", "usk_abc123", false},
		{"Bearer   usk_abc123  ", "usk_abc123", false},
		{"Basic abc123", "", true},
		{"", "", true},
		{"Bearer ", "", true},
	}

	for _, c := range cases {
		got, err := ExtractBearerToken(c.header)
		if c.wantErr {
			if err == nil {
				t.Errorf("ExtractBearerToken(%q): expected error, got nil", c.header)
			}
			continue
		}
		if err != nil {
			t.Errorf("ExtractBearerToken(%q): unexpected error: %v", c.header, err)
		}
		if got != c.want {
			t.Errorf("ExtractBearerToken(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}
