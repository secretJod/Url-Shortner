package handlers

import "testing"

func TestValidateURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"valid https", "https://example.com/path", "https://example.com/path", false},
		{"valid http", "http://example.com", "http://example.com", false},
		{"empty", "", "", true},
		{"missing scheme", "example.com", "", true},
		{"javascript scheme rejected", "javascript:alert(1)", "", true},
		{"data scheme rejected", "data:text/html,<script>alert(1)</script>", "", true},
		{"ftp scheme rejected", "ftp://example.com/file", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := validateURL(c.raw)
			if c.wantErr {
				if err == nil {
					t.Errorf("validateURL(%q): expected error, got nil (result %q)", c.raw, got)
				}
				return
			}
			if err != nil {
				t.Errorf("validateURL(%q): unexpected error: %v", c.raw, err)
			}
			if got != c.want {
				t.Errorf("validateURL(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}
