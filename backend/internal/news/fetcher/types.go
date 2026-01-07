package fetcher

import "time"

// Article represents a normalized news article from any source
type Article struct {
	Title           string                 `json:"title"`
	Summary         string                 `json:"summary"` // description
	URL             string                 `json:"url"`
	CanonicalURL    string                 `json:"canonical_url"`
	Source          string                 `json:"source"` // e.g., "naver", "newsapi"
	Publisher       string                 `json:"publisher,omitempty"`
	PublishedAt     time.Time              `json:"published_at"`
	FetchedAt       time.Time              `json:"fetched_at"`
	RawProviderData map[string]interface{} `json:"raw_provider_data,omitempty"`
}

type Fetcher interface {
	Fetch() ([]Article, error)
	Name() string
}
