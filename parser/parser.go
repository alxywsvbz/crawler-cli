package parser

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type PageData struct {
	Title string
	Links []string
}

func ExtractPageData(body io.Reader, baseURL *url.URL) (*PageData, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	data := &PageData{
		Links: make([]string, 0),
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && n.FirstChild != nil {
				data.Title = strings.TrimSpace(n.FirstChild.Data)
			}

			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						link := strings.TrimSpace(attr.Val)
						resolvedURL, err := baseURL.Parse(link)
						if err == nil {
							data.Links = append(data.Links, resolvedURL.String())
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return data, nil
}
