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

func createTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `<html><head><title>Root</title></head><body><a href="/p1">P1</a><a href="/p2">P2</a></body></html>`)
		case "/p1":
			io.WriteString(w, `<html><head><title>Page 1</title></head><body><a href="/p1/sub">Sub 1</a></body></html>`)
		case "/p2":
			io.WriteString(w, `<html><head><title>Page 2</title></head><body>Done</body></html>`)
		case "/p1/sub":
			io.WriteString(w, `<html><head><title>Subpage 1</title></head><body>Deepest</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestCrawler_Depth0(t *testing.T) {
	ts := createTestServer()
	defer ts.Close()

	cfg := &config.Config{
		URLs:           []string{ts.URL},
		Depth:          0,
		Timeout:        5 * time.Second,
		RequestTimeout: 2 * time.Second,
	}

	c := NewCrawler(cfg, log.New(io.Discard, "", 0))
	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// На depth=0 у корня не должно быть дочерних ссылок
	if len(results[0].Links) != 0 {
		t.Errorf("Expected 0 links at depth=0, got %d", len(results[0].Links))
	}
}

func TestCrawler_Depth1(t *testing.T) {
	ts := createTestServer()
	defer ts.Close()

	cfg := &config.Config{
		URLs:           []string{ts.URL},
		Depth:          1,
		Timeout:        5 * time.Second,
		RequestTimeout: 2 * time.Second,
	}

	c := NewCrawler(cfg, log.New(io.Discard, "", 0))
	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	root := results[0]
	// На depth=1 должны спарситься 2 дочерние ссылки 1-го уровня (/p1 и /p2)
	if len(root.Links) != 2 {
		t.Fatalf("Expected 2 links at depth=1, got %d", len(root.Links))
	}

	// У дочерних узлов не должно быть своих детей
	for _, child := range root.Links {
		if len(child.Links) != 0 {
			t.Errorf("Expected 0 nested links at depth=1 for %s, got %d", child.Resource, len(child.Links))
		}
	}
}

func TestCrawler_Depth2(t *testing.T) {
	ts := createTestServer()
	defer ts.Close()

	cfg := &config.Config{
		URLs:           []string{ts.URL},
		Depth:          2,
		Timeout:        5 * time.Second,
		RequestTimeout: 2 * time.Second,
	}

	c := NewCrawler(cfg, log.New(io.Discard, "", 0))
	results, err := c.Run(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	root := results[0]
	if len(root.Links) != 2 {
		t.Fatalf("Expected 2 root links at depth=2, got %d", len(root.Links))
	}

	// Находим /p1 и проверяем, что у него спарсился /p1/sub (2-й уровень)
	var p1Node *Node
	for _, child := range root.Links {
		if child.Title == "Page 1" {
			p1Node = child
			break
		}
	}

	if p1Node == nil {
		t.Fatal("Page 1 node not found")
	}

	if len(p1Node.Links) != 1 {
		t.Fatalf("Expected 1 nested link for Page 1 at depth=2, got %d", len(p1Node.Links))
	}

	if p1Node.Links[0].Title != "Subpage 1" {
		t.Errorf("Expected title 'Subpage 1', got '%s'", p1Node.Links[0].Title)
	}
}
