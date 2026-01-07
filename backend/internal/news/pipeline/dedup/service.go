package dedup

import (
	"crypto/sha256"
	"encoding/hex"

	"dx-unified/internal/news/fetcher"
	"dx-unified/internal/news/store/meili"
	"dx-unified/internal/shared/config"
)

type Service struct {
	cfg   *config.Config
	store *meili.Store
}

func NewService(cfg *config.Config, store *meili.Store) *Service {
	return &Service{cfg: cfg, store: store}
}

// IsDuplicate checks if the article is a duplicate.
// Simple impl: check hash set in memory? Or query Meili?
// For now, simple return false to unblock.
func (s *Service) IsDuplicate(a *fetcher.Article) (bool, string, float64) {
	// Logic to check URL hash or similarity
	// Returning false (unique) for now.
	return false, "", 0.0
}

func GenerateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
