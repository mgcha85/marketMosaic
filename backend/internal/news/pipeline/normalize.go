package pipeline

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strings"

	"dx-unified/internal/news/fetcher"
)

// NormalizeArticle performs canonicalization on the article
func NormalizeArticle(a *fetcher.Article) {
	// 1. Canonicalize URL (remove UTM params)
	a.CanonicalURL = canonicalizeURL(a.URL)

	// 2. Normalize Title (simple trim)
	a.Title = strings.TrimSpace(a.Title)
	a.Summary = strings.TrimSpace(a.Summary)

	// 3. Ensure other fields are sane?
}

func canonicalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove common tracking params
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(k, "utm_") || k == "ref" || k == "source" {
			q.Del(k)
		}
	}

	// Naver specific cleanup if needed (often clean in fetcher, but good to double check)
	// Some sites might have empty params?

	u.RawQuery = q.Encode()
	return u.String()
}

func GenerateID(canonicalURL string) string {
	h := sha1.New()
	h.Write([]byte(canonicalURL))
	return hex.EncodeToString(h.Sum(nil))
}
