package shortcode

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 61, 62, 63, 12345, 999999999, 18446744073709551615}
	for _, id := range cases {
		code := Encode(id)
		got, err := Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q) returned error: %v", code, err)
		}
		if got != id {
			t.Errorf("round trip mismatch: id=%d code=%q decoded=%d", id, code, got)
		}
	}
}

func TestEncodeIsDeterministicAndUnique(t *testing.T) {
	seen := make(map[string]uint64)
	for id := uint64(0); id < 10000; id++ {
		code := Encode(id)
		if prev, ok := seen[code]; ok {
			t.Fatalf("collision: id=%d and id=%d both encode to %q", prev, id, code)
		}
		seen[code] = id
	}
}

func TestDecodeInvalidCharacter(t *testing.T) {
	if _, err := Decode("abc!def"); err == nil {
		t.Error("expected error for invalid character, got nil")
	}
}
