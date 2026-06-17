package toml_test

import (
	"net/url"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestIssue1058_UnmarshalURL(t *testing.T) {
	type config struct {
		Endpoint *url.URL `toml:"endpoint"`
	}

	var cfg config
	err := toml.Unmarshal([]byte(`endpoint = "https://example.com/path"`), &cfg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if cfg.Endpoint == nil {
		t.Fatal("expected endpoint URL to be set")
	}
	if got, want := cfg.Endpoint.String(), "https://example.com/path"; got != want {
		t.Fatalf("Endpoint = %q, want %q", got, want)
	}
}

func TestIssue1058_UnmarshalURLValue(t *testing.T) {
	type config struct {
		Endpoint url.URL `toml:"endpoint"`
	}

	var cfg config
	err := toml.Unmarshal([]byte(`endpoint = "https://example.com/"`), &cfg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got, want := cfg.Endpoint.String(), "https://example.com/"; got != want {
		t.Fatalf("Endpoint = %q, want %q", got, want)
	}
}
