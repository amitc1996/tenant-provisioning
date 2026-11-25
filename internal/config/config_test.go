package config

import "testing"

func TestLoadDefaultHTTPAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")

	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.HTTPAddr)
	}
}

func TestLoadCustomHTTPAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")

	cfg := Load()

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.HTTPAddr)
	}
}
