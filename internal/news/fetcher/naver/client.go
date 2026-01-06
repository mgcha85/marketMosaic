package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dx-unified/internal/news/config"
	"dx-unified/internal/news/fetcher"
	"golang.org/x/time/rate"
)

const (
	BaseURL = "https://openapi.naver.com/v1/search/news.json"
	Source  = "naver_search_api"
)

type Client struct {
	cfg         *config.Config
	rateLimiter *rate.Limiter
	httpClient  *http.Client
}

type Response struct {
	LastBuildDate string `json:"lastBuildDate"`
	Total         int    `json:"total"`
	Start         int    `json:"start"`
	Display       int    `json:"display"`
	Items         []Item `json:"items"`
}

type Item struct {
	Title        string `json:"title"`
	Originallink string `json:"originallink"`
	Link         string `json:"link"`
	Description  string `json:"description"`
	PubDate      string `json:"pubDate"`
}

func New(cfg *config.Config) *Client {
	// 9 QPS limit as per requirements (conservative)
	limiter := rate.NewLimiter(rate.Limit(9), 1)

	return &Client{
		cfg:         cfg,
		rateLimiter: limiter,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Name() string {
	return Source
}

func (c *Client) Fetch() ([]fetcher.Article, error) {
	var allArticles []fetcher.Article

	for _, query := range c.cfg.NaverQueries {
		if err := c.rateLimiter.Wait(context.Background()); err != nil { // Wait needs a context
			return nil, fmt.Errorf("rate limiter wait: %w", err)
		}

		articles, err := c.fetchQuery(query)
		if err != nil {
			// Log error but continue with other queries?
			// For now, let's just print/log and continue, gathering partial results is better than none.
			fmt.Printf("Error fetching query %s: %v\n", query, err)
			continue
		}
		allArticles = append(allArticles, articles...)
	}

	return allArticles, nil
}

func (c *Client) fetchQuery(query string) ([]fetcher.Article, error) {
	// Sort by date to get latest news
	u, _ := url.Parse(BaseURL)
	q := u.Query()
	q.Set("query", query)
	q.Set("display", "100") // Max allowed
	q.Set("sort", "date")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Naver-Client-Id", c.cfg.NaverClientID)
	req.Header.Set("X-Naver-Client-Secret", c.cfg.NaverClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("naver api error: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var articles []fetcher.Article
	for _, item := range result.Items {
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			// Try another format if needed or just skip/log
			pubDate = time.Now() // Fallback
		}

		// Prefer original link if available
		link := item.Originallink
		if link == "" {
			link = item.Link
		}

		articles = append(articles, fetcher.Article{
			Title:       stripHTML(item.Title),
			Summary:     stripHTML(item.Description),
			URL:         link,
			Source:      Source,
			PublishedAt: pubDate,
			FetchedAt:   time.Now(),
			RawProviderData: map[string]interface{}{
				"original_title": item.Title,
				"description":    item.Description,
			},
		})
	}

	return articles, nil
}

func stripHTML(s string) string {
	// Simple replacement for now.
	// For production, might want a robust HTML stripper, but for <b> tags, strings.Replace is fast.
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}
