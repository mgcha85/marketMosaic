package pipeline

import (
	"fmt"
	"log"
	"time"

	"dx-unified/internal/news/fetcher"
	"dx-unified/internal/news/pipeline/dedup"
	"dx-unified/internal/news/store/meili"
	"dx-unified/internal/shared/config"

	"github.com/google/uuid"
)

type Processor struct {
	cfg      *config.Config
	fetchers []fetcher.Fetcher
	filter   *Filter
	dedupSvc *dedup.Service
	store    *meili.Store
}

func NewProcessor(cfg *config.Config, store *meili.Store, fetchers []fetcher.Fetcher) *Processor {
	return &Processor{
		cfg:      cfg,
		fetchers: fetchers,
		store:    store,
		filter:   NewFilter(cfg),
		dedupSvc: dedup.NewService(cfg, store),
	}
}

func (p *Processor) Run() {
	runID := uuid.New().String()
	startTime := time.Now()

	log.Printf("[Run %s] Started", runID)

	stats := make(map[string]interface{})
	var allErrors []string

	totalFetched := 0
	// totalFiltered := 0
	totalStored := 0
	totalDups := 0

	for _, f := range p.fetchers {
		log.Printf("[Run %s] Fetching from %s...", runID, f.Name())
		articles, err := f.Fetch()
		if err != nil {
			msg := fmt.Sprintf("Fetcher %s error: %v", f.Name(), err)
			log.Println(msg)
			allErrors = append(allErrors, msg)
			continue
		}

		sourceFetched := len(articles)
		totalFetched += sourceFetched

		var docsToSave []meili.ArticleDoc

		for _, a := range articles {
			// 1. Normalize
			NormalizeArticle(&a)
			// passed, reason := p.filter.IsEconomicOrStockNews(&a)
			// if !passed {
			// 	log.Printf("[Run %s] Filtered out article '%s': %s", runID, a.Title, reason)
			// 	continue
			// }

			// 3. Dedup
			isDup, dupOf, score := p.dedupSvc.IsDuplicate(&a)
			dupState := "unique"
			if isDup {
				dupState = "duplicate"
				totalDups++
			}

			// 4. Transform to Doc
			doc := meili.ArticleDoc{
				ID:           GenerateID(a.CanonicalURL),
				Title:        a.Title,
				Summary:      a.Summary,
				URL:          a.URL,
				CanonicalURL: a.CanonicalURL,
				Source:       a.Source,
				Publisher:    a.Publisher,
				PublishedAt:  a.PublishedAt,
				FetchedAt:    a.FetchedAt,
				DupState:     dupState,
				DupOf:        dupOf,
				DupScore:     score,
				Tags:         []string{}, // TODO: tagger
			}

			// If Duplicate, we might still save it with dup_state=duplicate (as per plan/requirements: "Record discarded items in runs or store with duplicate state")
			// Let's save it for traceability.
			docsToSave = append(docsToSave, doc)
			totalStored++
		}

		if err := p.store.SaveArticles(docsToSave); err != nil {
			msg := fmt.Sprintf("Store error for %s: %v", f.Name(), err)
			log.Println(msg)
			allErrors = append(allErrors, msg)
		}

		stats[f.Name()] = map[string]int{
			"fetched": sourceFetched,
			"saved":   len(docsToSave),
		}
	}

	endTime := time.Now()
	status := "success"
	if len(allErrors) > 0 {
		status = "partial_failure" // or failed
	}

	runLog := &meili.RunLog{
		RunID:     runID,
		StartedAt: startTime,
		EndedAt:   endTime,
		Status:    status,
		Errors:    allErrors,
		Stats: map[string]interface{}{
			"total_fetched": totalFetched,
			"total_stored":  totalStored,
			"total_dups":    totalDups,
			"sources":       stats,
		},
	}

	if err := p.store.SaveRun(runLog); err != nil {
		log.Printf("[Run %s] Failed to save run log: %v", runID, err)
	}

	log.Printf("[Run %s] Finished in %v. Stored: %d, Dups: %d", runID, endTime.Sub(startTime), totalStored, totalDups)
}
