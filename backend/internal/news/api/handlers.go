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

		// Migration
		news.POST("/migration", h.IngestArticles)
	}
}

// GetArticles returns a list of articles
func (h *Handler) GetArticles(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	source := c.Query("source")
	keyword := c.Query("keyword")
	dateTo := c.Query("date_to") // Expecting YYYYMMDD or ISO

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	filter := []string{}
	if source != "" {
		filter = append(filter, "source = \""+source+"\"")
	}
	if dateTo != "" {
		// Assume Meilisearch stores published_at as ISO string or timestamp
		// If input is YYYYMMDD, convert to ISO end of day?
		// For simplicity, assume User sends valid format or we check string comparison
		// If stored as "2024-01-01T..." and input is "20240101", comparison might fail.
		// However, frontend Time Travel sends YYYYMMDD usually.
		// Let's assume stored is ISO.
		// If dateTo is 8 chars, format to YYYY-MM-DD
		if len(dateTo) == 8 {
			dateTo = dateTo[:4] + "-" + dateTo[4:6] + "-" + dateTo[6:]
		}
		// Less than or equal to end of that day
		filter = append(filter, "published_at <= \""+dateTo+"T23:59:59Z\"")
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
	dateTo := c.Query("date_to")

	limit, _ := strconv.Atoi(limitStr)

	filter := []string{}
	if dateTo != "" {
		if len(dateTo) == 8 {
			dateTo = dateTo[:4] + "-" + dateTo[4:6] + "-" + dateTo[6:]
		}
		filter = append(filter, "published_at <= \""+dateTo+"T23:59:59Z\"")
	}

	// Allow empty query to list all/recent articles
	// Meilisearch treats empty string as placeholder search
	searchReq := &meilisearch.SearchRequest{
		Limit: int64(limit),
		Sort:  []string{"published_at:desc"},
	}

	if len(filter) > 0 {
		searchReq.Filter = filter
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

// IngestArticles handles batch ingestion of news articles
func (h *Handler) IngestArticles(c *gin.Context) {
	var articles []meili.ArticleDoc // Using meili package directly for now as store expects it
	if err := c.ShouldBindJSON(&articles); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SaveArticles(articles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Articles ingested successfully",
		"count":   len(articles),
	})
}
