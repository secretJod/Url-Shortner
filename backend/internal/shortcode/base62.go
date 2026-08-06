// Package shortcode turns numeric IDs (from the Redis INCR counter) into
// short, URL-safe base62 codes, and back again.
//
// Base62 alphabet = [0-9a-zA-Z], so codes are compact and collision-free
// by construction: every distinct counter value maps to exactly one code.
package shortcode

import (
	"errors"
	"strings"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const base = uint64(len(alphabet))

var ErrInvalidCode = errors.New("shortcode: invalid character in code")

// Encode converts a positive integer ID into a base62 short code.
// e.g. 0 -> "0", 61 -> "Z", 62 -> "10"
func Encode(id uint64) string {
	if id == 0 {
		return string(alphabet[0])
	}

	var sb strings.Builder
	// Encode digits least-significant-first, then reverse.
	digits := make([]byte, 0, 11) // enough for a uint64
	for id > 0 {
		digits = append(digits, alphabet[id%base])
		id /= base
	}
	for i := len(digits) - 1; i >= 0; i-- {
		sb.WriteByte(digits[i])
	}
	return sb.String()
}

// Decode converts a base62 short code back into its numeric ID.
// Useful for debugging/admin tooling; not needed on the hot path.
func Decode(code string) (uint64, error) {
	var id uint64
	for _, ch := range code {
		idx := strings.IndexRune(alphabet, ch)
		if idx < 0 {
			return 0, ErrInvalidCode
		}
		id = id*base + uint64(idx)
	}
	return id, nil
}
