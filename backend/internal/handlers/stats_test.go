package handlers

import (
	"testing"

	"github.com/yourorg/urlshortener/internal/store"
)

func TestOwnsLink(t *testing.T) {
	ownerID := uint64(42)
	otherID := uint64(99)

	cases := []struct {
		name   string
		link   *store.Link
		apiKey *store.ApiKey
		want   bool
	}{
		{
			name:   "owner matches",
			link:   &store.Link{UserID: &ownerID},
			apiKey: &store.ApiKey{UserID: ownerID},
			want:   true,
		},
		{
			name:   "different user",
			link:   &store.Link{UserID: &ownerID},
			apiKey: &store.ApiKey{UserID: otherID},
			want:   false,
		},
		{
			name:   "anonymous link",
			link:   &store.Link{UserID: nil},
			apiKey: &store.ApiKey{UserID: ownerID},
			want:   false,
		},
		{
			name:   "no api key",
			link:   &store.Link{UserID: &ownerID},
			apiKey: nil,
			want:   false,
		},
		{
			name:   "anonymous link, no api key",
			link:   &store.Link{UserID: nil},
			apiKey: nil,
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ownsLink(c.link, c.apiKey); got != c.want {
				t.Errorf("ownsLink() = %v, want %v", got, c.want)
			}
		})
	}
}
