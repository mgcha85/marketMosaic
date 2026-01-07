package newsapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"dx-unified/internal/news/fetcher"
	"dx-unified/internal/shared/config"
)

const (
	BaseURL = "https://newsapi.org/v2/everything"
	Source  = "newsapi"
)

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

type Response struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []Item `json:"articles"`
	Code         string `json:"code"`
	Message      string `json:"message"`
}

type Item struct {
	Source      SourceInfo `json:"source"`
	Author      string     `json:"author"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Url         string     `json:"url"`
	UrlToImage  string     `json:"urlToImage"`
	PublishedAt string     `json:"publishedAt"`
	Content     string     `json:"content"`
}

type SourceInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Name() string {
	return Source
}

func (c *Client) Fetch() ([]fetcher.Article, error) {
	// Strategy: Use /v2/everything with English keywords to fetch US/Global business news.
	// We stay within 100 reqs/day by running every 15 minutes (96 reqs/day).
	// Keywords are selected to maximize economic/market news coverage.

	u, _ := url.Parse(BaseURL)
	q := u.Query()
	q.Set("q", "economy OR stock OR market OR finance OR business")
	q.Set("language", "en")
	q.Set("sortBy", "publishedAt")
	q.Set("pageSize", "100")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", c.cfg.NewsAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("newsapi error: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "ok" {
		return nil, fmt.Errorf("newsapi error: %s - %s", result.Code, result.Message)
	}

	var articles []fetcher.Article
	for _, item := range result.Articles {
		pubDate, err := time.Parse(time.RFC3339, item.PublishedAt)
		if err != nil {
			pubDate = time.Now()
		}

		articles = append(articles, fetcher.Article{
			Title:       item.Title,
			Summary:     item.Description,
			URL:         item.Url,
			Source:      Source,
			Publisher:   item.Source.Name,
			PublishedAt: pubDate,
			FetchedAt:   time.Now(),
			RawProviderData: map[string]interface{}{
				"author":  item.Author,
				"content": item.Content,
			},
		})
	}

	return articles, nil
}
