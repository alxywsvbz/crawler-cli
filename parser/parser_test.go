package parser

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtractPageData(t *testing.T) {
	htmlData := `
	<!DOCTYPE html>
	<html>
	<head><title>Test Page</title></head>
	<body>
		<a href="/about">About</a>
		<a href="https://example.com/contact">Contact</a>
		<a href="invalid-url-%%">Invalid</a>
	</body>
	</html>`

	baseURL, _ := url.Parse("https://example.com")
	data, err := ExtractPageData(strings.NewReader(htmlData), baseURL)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if data.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", data.Title)
	}

	expectedLinks := map[string]bool{
		"https://example.com/about":   true,
		"https://example.com/contact": true,
	}

	for _, link := range data.Links {
		if !expectedLinks[link] && link != "https://example.com/invalid-url-%%" {
			t.Errorf("Unexpected link found: %s", link)
		}
	}
}
