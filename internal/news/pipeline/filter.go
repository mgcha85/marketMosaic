package pipeline

import (
	"fmt"
	"strings"

	"dx-unified/internal/news/config"
	"dx-unified/internal/news/fetcher"
)

type Filter struct {
	cfg *config.Config
}

func NewFilter(cfg *config.Config) *Filter {
	return &Filter{cfg: cfg}
}

func (f *Filter) IsEconomicOrStockNews(a *fetcher.Article) (bool, string) {
	// 1. Check Source constraint? (Already done in Fetcher for NewsAPI)
	// 2. Keyword check in Title/Summary

	searchContent := a.Title + " " + a.Summary
	if matched, keyword := f.containsAny(searchContent, f.cfg.EconKeywordsAllow); matched {
		return true, fmt.Sprintf("matched keyword: %s", keyword)
	}
	return false, fmt.Sprintf("no keyword match in title/summary. Content: %s...", limitString(searchContent, 50))
}

func (f *Filter) containsAny(text string, keywords []string) (bool, string) {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true, k
		}
	}
	return false, ""
}

func limitString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
