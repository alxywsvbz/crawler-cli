package crawler

import (
	"context"
	"crawler-cli/config"
	"crawler-cli/parser"
	"crawler-cli/storage"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Node struct {
	Resource string  `json:"resource"`
	Title    string  `json:"title"`
	Links    []*Node `json:"links"`
}

type Crawler struct {
	cfg        *config.Config
	client     *http.Client
	visited    *storage.VisitedMap
	semaphore  chan struct{}
	fileLogger *log.Logger
}

func NewCrawler(cfg *config.Config, logger *log.Logger) *Crawler {
	// Настройка HTTP-клиента: запрет автоматических редиректов
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Crawler{
		cfg:        cfg,
		client:     client,
		visited:    storage.NewVisitedMap(),
		semaphore:  make(chan struct{}, 10), // Ограничение: максимум 10 одновременных запросов
		fileLogger: logger,
	}
}

func (c *Crawler) Run(ctx context.Context) ([]*Node, error) {
	var wg sync.WaitGroup
	results := make([]*Node, len(c.cfg.URLs))

	for i, startURL := range c.cfg.URLs {
		wg.Add(1)
		go func(idx int, rawURL string) {
			defer wg.Done()

			parsedURL, err := url.Parse(rawURL)
			if err != nil {
				c.fileLogger.Printf("[ERROR] Invalid start URL %s: %v", rawURL, err)
				return
			}

			if !c.visited.TryVisit(rawURL) {
				return
			}

			results[idx] = c.crawlNode(ctx, rawURL, parsedURL.Host, 1)
		}(i, startURL)
	}

	wg.Wait()

	// Фильтрация nil элементов (если стартовый URL оказался невалидным или уже посещённым)
	finalResults := make([]*Node, 0, len(results))
	for _, res := range results {
		if res != nil {
			finalResults = append(finalResults, res)
		}
	}

	return finalResults, nil
}

func (c *Crawler) crawlNode(ctx context.Context, currentURL string, allowedHost string, currentDepth int) *Node {
	node := &Node{
		Resource: currentURL,
		Links:    make([]*Node, 0),
	}

	// Захватом семафора контролируем количество параллельных goroutine на HTTP-запросы
	select {
	case <-ctx.Done():
		c.fileLogger.Printf("[CANCELLED] Context done before fetching %s", currentURL)
		return node
	case c.semaphore <- struct{}{}:
	}

	pageData, err := c.fetchAndParse(ctx, currentURL)
	<-c.semaphore // Освобождаем слот семафора

	if err != nil {
		c.fileLogger.Printf("[FETCH ERROR] %s: %v", currentURL, err)
		return node
	}

	node.Title = pageData.Title

	// Проверка достижения максимальной глубины
	if currentDepth >= c.cfg.Depth {
		return node
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, link := range pageData.Links {
		parsedLink, err := url.Parse(link)
		if err != nil {
			continue
		}

		// Фильтрация: только ссылки в пределах домена стартового URL
		if parsedLink.Host != allowedHost {
			continue
		}

		// Предотвращение циклов и повторных запросов
		if !c.visited.TryVisit(link) {
			continue
		}

		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()
			childNode := c.crawlNode(ctx, targetURL, allowedHost, currentDepth+1)
			if childNode != nil {
				mu.Lock()
				node.Links = append(node.Links, childNode)
				mu.Unlock()
			}
		}(link)
	}

	wg.Wait()
	return node
}

func (c *Crawler) fetchAndParse(ctx context.Context, targetURL string) (*parser.PageData, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.fileLogger.Printf("[HTTP STATUS] %s -> %d", targetURL, resp.StatusCode)

	// Пропуск редиректов (3xx) и не-OK статусов
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("redirect skipped (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	// Пропуск не-HTML ресурсов
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("skipped non-HTML content-type: %s", contentType)
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	return parser.ExtractPageData(resp.Body, parsedURL)
}

func SaveJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
