package meili

import (
	"time"
)

type ArticleDoc struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	URL          string    `json:"url"`
	CanonicalURL string    `json:"canonical_url"`
	Source       string    `json:"source"`
	Publisher    string    `json:"publisher"`
	PublishedAt  time.Time `json:"published_at"`
	FetchedAt    time.Time `json:"fetched_at"`

	DupState string  `json:"dup_state"` // unique, duplicate
	DupOf    string  `json:"dup_of,omitempty"`
	DupScore float64 `json:"dup_score,omitempty"`

	Tags []string `json:"tags,omitempty"`
}

type RunLog struct {
	RunID     string                 `json:"run_id"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at"`
	Status    string                 `json:"status"` // success, failed
	Stats     map[string]interface{} `json:"stats"`
	Errors    []string               `json:"errors,omitempty"`
}

func (s *Store) SaveArticles(articles []ArticleDoc) error {
	if len(articles) == 0 {
		return nil
	}
	_, err := s.Client.Index(IndexArticles).AddDocuments(articles, nil)
	return err
}

func (s *Store) SaveRun(run *RunLog) error {
	_, err := s.Client.Index(IndexRuns).AddDocuments([]*RunLog{run}, nil)
	return err
}
