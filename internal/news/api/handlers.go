package api

import (
	"net/http"
	"strconv"

	"dx-unified/internal/news/store/meili"

	"github.com/gin-gonic/gin"
	"github.com/meilisearch/meilisearch-go"
)

// Handler holds dependencies for News API handlers
type Handler struct {
	store *meili.Store
}

// NewHandler creates a new News API handler
func NewHandler(store *meili.Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registers all News API routes under /news prefix
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	news := rg.Group("/news")
	{
		news.GET("/articles", h.GetArticles)
		news.GET("/articles/:id", h.GetArticle)
		news.GET("/runs", h.GetRuns)
		news.GET("/search", h.SearchArticles)
	}
}

// GetArticles returns a list of articles
func (h *Handler) GetArticles(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	source := c.Query("source")
	keyword := c.Query("keyword")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	filter := []string{}
	if source != "" {
		filter = append(filter, "source = \""+source+"\"")
	}

	searchReq := &meilisearch.SearchRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
		Sort:   []string{"published_at:desc"},
	}

	if len(filter) > 0 {
		searchReq.Filter = filter
	}

	query := ""
	if keyword != "" {
		query = keyword
	}

	result, err := h.store.Client.Index(meili.IndexArticles).Search(query, searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    result.EstimatedTotalHits,
		"count":    len(result.Hits),
		"offset":   offset,
		"articles": result.Hits,
	})
}

// GetArticle returns a single article by ID
func (h *Handler) GetArticle(c *gin.Context) {
	id := c.Param("id")

	var article map[string]interface{}
	err := h.store.Client.Index(meili.IndexArticles).GetDocument(id, nil, &article)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	c.JSON(http.StatusOK, article)
}

// SearchArticles performs full-text search on articles
func (h *Handler) SearchArticles(c *gin.Context) {
	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	searchReq := &meilisearch.SearchRequest{
		Limit: int64(limit),
		Sort:  []string{"published_at:desc"},
	}

	result, err := h.store.Client.Index(meili.IndexArticles).Search(query, searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":    query,
		"total":    result.EstimatedTotalHits,
		"count":    len(result.Hits),
		"articles": result.Hits,
	})
}

// GetRuns returns batch run logs
func (h *Handler) GetRuns(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	searchReq := &meilisearch.SearchRequest{
		Limit: int64(limit),
		Sort:  []string{"started_at:desc"},
	}

	result, err := h.store.Client.Index(meili.IndexRuns).Search("", searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(result.Hits),
		"runs":  result.Hits,
	})
}
