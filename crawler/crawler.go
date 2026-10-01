package crawler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"crawler-cli/config"
	"crawler-cli/parser"
	"crawler-cli/storage"
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
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Crawler{
		cfg:        cfg,
		client:     client,
		visited:    storage.NewVisitedMap(),
		semaphore:  make(chan struct{}, 10), // Максимум 10 одновременно работающих горутин
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

			// Корневой обход начинается с глубины 0
			results[idx] = c.crawlNode(ctx, rawURL, parsedURL.Host, 0)
		}(i, startURL)
	}

	wg.Wait()

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

	pageData, err := c.fetchAndParse(ctx, currentURL)
	if err != nil {
		c.fileLogger.Printf("[FETCH ERROR] %s: %v", currentURL, err)
		return node
	}

	node.Title = pageData.Title

	// Прекращаем обход дочерних ссылок, если достигли максимальной глубины
	if currentDepth >= c.cfg.Depth {
		return node
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, link := range pageData.Links {
		parsedLink, err := url.Parse(link)
		if err != nil || parsedLink.Host != allowedHost {
			continue
		}

		if !c.visited.TryVisit(link) {
			continue
		}

		// Контроль количества горутин: захватываем семафор ПЕРЕД созданием горутины
		select {
		case <-ctx.Done():
			c.fileLogger.Printf("[CANCELLED] Context done before queueing %s", link)
			return node
		case c.semaphore <- struct{}{}:
		}

		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()
			defer func() { <-c.semaphore }() // Освобождаем семафор по завершении горутины

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

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("redirect skipped (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

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
