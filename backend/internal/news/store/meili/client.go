package meili

import (
	"fmt"
	"log"

	"github.com/meilisearch/meilisearch-go"
)

const (
	IndexArticles = "articles"
	IndexRuns     = "runs"
)

type Store struct {
	Client meilisearch.ServiceManager
}

func New(host, apiKey string) (*Store, error) {
	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	s := &Store{Client: client}
	if err := s.EnsureIndexes(); err != nil {
		return nil, fmt.Errorf("failed to ensure indexes: %w", err)
	}

	return s, nil
}

func (s *Store) EnsureIndexes() error {
	// Articles Index
	if _, err := s.Client.GetIndex(IndexArticles); err != nil {
		log.Printf("Creating index: %s", IndexArticles)
		_, err := s.Client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        IndexArticles,
			PrimaryKey: "id",
		})
		if err != nil {
			return err
		}
	}

	// Wait a bit for index creation (async) or check task status in real implementation
	// For simplicity, we assume it's fast enough or eventually consistent for setting settings

	// Articles Settings
	articleSettings := &meilisearch.Settings{
		SearchableAttributes: []string{
			"title",
			"summary",
			"tags",
		},
		FilterableAttributes: []string{
			"source",
			"published_at",
			"dup_state",
			"tags",
		},
		SortableAttributes: []string{
			"published_at",
			"fetched_at",
			"market_relevance_score",
		},
	}
	if _, err := s.Client.Index(IndexArticles).UpdateSettings(articleSettings); err != nil {
		return fmt.Errorf("failed to update articles settings: %w", err)
	}

	// Runs Index
	if _, err := s.Client.GetIndex(IndexRuns); err != nil {
		log.Printf("Creating index: %s", IndexRuns)
		_, err := s.Client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        IndexRuns,
			PrimaryKey: "run_id",
		})
		if err != nil {
			return err
		}
	}

	// Runs Settings
	runsSettings := &meilisearch.Settings{
		FilterableAttributes: []string{
			"status",
			"started_at",
		},
		SortableAttributes: []string{
			"started_at",
			"ended_at",
		},
	}
	if _, err := s.Client.Index(IndexRuns).UpdateSettings(runsSettings); err != nil {
		return fmt.Errorf("failed to update runs settings: %w", err)
	}

	return nil
}
