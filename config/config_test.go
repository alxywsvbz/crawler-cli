package config

import (
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	args := []string{
		"--urls", "https://example.com,https://google.com",
		"--depth", "2",
		"--timeout", "30s",
		"--request-timeout", "5s",
		"--output", "out.json",
		"--log", "test.log",
	}

	cfg, err := ParseFlags(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(cfg.URLs) != 2 || cfg.URLs[0] != "https://example.com" {
		t.Errorf("Unexpected URLs: %v", cfg.URLs)
	}
	if cfg.Depth != 2 {
		t.Errorf("Expected depth 2, got %d", cfg.Depth)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", cfg.Timeout)
	}
}

func TestParseFlags_EmptyURLs(t *testing.T) {
	_, err := ParseFlags([]string{"--depth", "2"})
	if err == nil {
		t.Error("Expected error for empty urls, got nil")
	}
}
