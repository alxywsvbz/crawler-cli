package crawler

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crawler-cli/config"
)

func TestCrawler_DomainBoundariesAndDepth(t *testing.T) {
	// Создание тестового HTTP-сервера
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `<html><head><title>Home</title></head><body><a href="/page1">Page 1</a><a href="https://external.com">External</a></body></html>`)
		case "/page1":
			io.WriteString(w, `<html><head><title>Page 1</title></head><body><a href="/page2">Page 2</a></body></html>`)
		case "/page2":
			io.WriteString(w, `<html><head><title>Page 2</title></head><body>Done</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		URLs:           []string{server.URL},
		Depth:          2,
		Timeout:        5 * time.Second,
		RequestTimeout: 2 * time.Second,
	}

	logger := log.New(io.Discard, "", 0)
	c := NewCrawler(cfg, logger)

	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatalf("Crawler failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 root node, got %d", len(results))
	}

	root := results[0]
	if root.Title != "Home" {
		t.Errorf("Expected root title 'Home', got '%s'", root.Title)
	}

	if len(root.Links) != 1 {
		t.Fatalf("Expected 1 internal link, got %d", len(root.Links))
	}

	if root.Links[0].Title != "Page 1" {
		t.Errorf("Expected child title 'Page 1', got '%s'", root.Links[0].Title)
	}

	// Проверка ограничения глубины (depth 2 не должен заходить на /page2)
	if len(root.Links[0].Links) != 0 {
		t.Errorf("Expected 0 links at depth 2, got %d", len(root.Links[0].Links))
	}
}
