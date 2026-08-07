package config

import (
	"net/http"
	"testing"
)

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"kosong", "", nil},
		{"hanya spasi", "   ", nil},
		{"satu origin", "http://localhost:3000", []string{"http://localhost:3000"}},
		{
			"beberapa origin dengan spasi",
			"http://localhost:3000, https://cms.example.com",
			[]string{"http://localhost:3000", "https://cms.example.com"},
		},
		{"trailing slash dibuang", "http://localhost:3000/", []string{"http://localhost:3000"}},
		{"entri kosong dilewati", "http://localhost:3000,,", []string{"http://localhost:3000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOrigins(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("parseOrigins(%q) = %v, mau %v", tt.raw, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d = %q, mau %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSameSiteMode(t *testing.T) {
	tests := []struct {
		value string
		want  http.SameSite
	}{
		{"none", http.SameSiteNoneMode},
		{"lax", http.SameSiteLaxMode},
		{"strict", http.SameSiteStrictMode},
		{"", http.SameSiteStrictMode},       // default aman
		{"ngawur", http.SameSiteStrictMode}, // nilai tak dikenal → Strict
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			cfg := &Config{CookieSameSite: tt.value}
			if got := cfg.SameSiteMode(); got != tt.want {
				t.Errorf("SameSiteMode(%q) = %v, mau %v", tt.value, got, tt.want)
			}
		})
	}
}
