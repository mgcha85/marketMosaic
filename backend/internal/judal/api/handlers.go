package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"dx-unified/internal/judal/crawler"
	"dx-unified/internal/judal/database"
	"dx-unified/internal/judal/models"

	"github.com/gin-gonic/gin"
)

var (
	crawlerInstance *crawler.Crawler
	crawlMutex      sync.Mutex
	isCrawling      bool
	lastCrawlResult *crawler.CrawlResult
)

// ensureCrawler lazily creates the crawler instance
func ensureCrawler() *crawler.Crawler {
	if crawlerInstance == nil {
		crawlerInstance = crawler.NewCrawler(1500 * time.Millisecond)
	}
	return crawlerInstance
}

// Handler holds dependencies for Judal API handlers
type Handler struct{}

// NewHandler creates a new Judal API handler
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers all Judal API routes under /judal prefix
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	judal := rg.Group("/judal")
	{
		// 테마 관련 (DB에서 조회)
		judal.GET("/themes", h.GetThemes)
		judal.GET("/themes/:themeIdx", h.GetTheme)
		judal.GET("/themes/:themeIdx/stocks", h.GetThemeStocks)

		// 종목 관련 (DB에서 조회)
		judal.GET("/stocks", h.GetStocks)
		judal.GET("/stocks/:code", h.GetStock)
		judal.GET("/stocks/:code/themes", h.GetStockThemes)
		judal.GET("/stocks/:code/history", h.GetStockHistory)

		// 실시간 크롤링 API
		judal.GET("/realtime/tabs", h.GetAvailableTabs)
		judal.GET("/realtime/themes/:tab", h.RealtimeThemeTab)
		judal.GET("/realtime/stocks/:tab", h.RealtimeStockTab)

		// 크롤링 제어
		judal.POST("/crawl", h.TriggerCrawl)
		judal.POST("/crawl/batch", h.TriggerBatchCrawl)
		judal.GET("/status", h.GetStatus)

		// 히스토리 및 로그
		judal.GET("/history/dates", h.GetHistoryDates)
		judal.GET("/logs", h.GetCrawlLogs)

		// Migration
		judal.POST("/migration/themes", h.IngestThemes)
		judal.POST("/migration/stocks", h.IngestStocks)
	}
}

// GetThemes 전체 테마 목록 조회
func (h *Handler) GetThemes(c *gin.Context) {
	repo := database.NewRepository()
	themes, err := repo.GetAllThemes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(themes),
		"themes": themes,
	})
}

// GetTheme 특정 테마 조회
func (h *Handler) GetTheme(c *gin.Context) {
	themeIdxStr := c.Param("themeIdx")
	themeIdx, err := strconv.Atoi(themeIdxStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme index"})
		return
	}

	repo := database.NewRepository()
	theme, err := repo.GetThemeByIdx(themeIdx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
		return
	}

	c.JSON(http.StatusOK, theme)
}

// GetThemeStocks 테마별 종목 목록 조회
func (h *Handler) GetThemeStocks(c *gin.Context) {
	themeIdxStr := c.Param("themeIdx")
	themeIdx, err := strconv.Atoi(themeIdxStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme index"})
		return
	}

	repo := database.NewRepository()

	theme, err := repo.GetThemeByIdx(themeIdx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
		return
	}

	stocks, err := repo.GetStocksByTheme(themeIdx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var stocksJSON []interface{}
	for _, s := range stocks {
		stocksJSON = append(stocksJSON, s.ToJSON())
	}

	c.JSON(http.StatusOK, gin.H{
		"theme":  theme,
		"count":  len(stocks),
		"stocks": stocksJSON,
	})
}

// GetStocks 종목 목록 조회
func (h *Handler) GetStocks(c *gin.Context) {
	params := database.StockQueryParams{
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
		Market: c.Query("market"),
	}

	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			params.Limit = v
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			params.Offset = v
		}
	}

	repo := database.NewRepository()
	stocks, total, err := repo.GetStocks(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var stocksJSON []interface{}
	for _, s := range stocks {
		stocksJSON = append(stocksJSON, s.ToJSON())
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  total,
		"count":  len(stocks),
		"offset": params.Offset,
		"stocks": stocksJSON,
	})
}

// GetStock 특정 종목 조회
func (h *Handler) GetStock(c *gin.Context) {
	code := c.Param("code")

	repo := database.NewRepository()
	stock, err := repo.GetStockByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	themes, _ := repo.GetThemesByStock(code)

	stockJSON := stock.ToJSON()
	var themeNames []string
	for _, t := range themes {
		themeNames = append(themeNames, t.Name)
	}
	stockJSON.RelatedThemes = themeNames

	c.JSON(http.StatusOK, stockJSON)
}

// GetStockThemes 종목이 속한 테마 조회
func (h *Handler) GetStockThemes(c *gin.Context) {
	code := c.Param("code")

	repo := database.NewRepository()
	themes, err := repo.GetThemesByStock(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stock_code": code,
		"count":      len(themes),
		"themes":     themes,
	})
}

// TriggerCrawl 크롤링 시작
func (h *Handler) TriggerCrawl(c *gin.Context) {
	crawlMutex.Lock()
	if isCrawling {
		crawlMutex.Unlock()
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Crawling already in progress",
			"message": "Please wait for the current crawl to complete",
		})
		return
	}
	isCrawling = true
	crawlMutex.Unlock()

	go func() {
		defer func() {
			crawlMutex.Lock()
			isCrawling = false
			crawlMutex.Unlock()
		}()

		result, _ := ensureCrawler().CrawlAll()

		crawlMutex.Lock()
		lastCrawlResult = result
		crawlMutex.Unlock()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Crawling started",
		"status":  "Check /judal/status for progress",
	})
}

// TriggerBatchCrawl 일배치 크롤링 시작
func (h *Handler) TriggerBatchCrawl(c *gin.Context) {
	crawlMutex.Lock()
	if isCrawling {
		crawlMutex.Unlock()
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Crawling already in progress",
			"message": "Please wait for the current crawl to complete",
		})
		return
	}
	isCrawling = true
	crawlMutex.Unlock()

	go func() {
		defer func() {
			crawlMutex.Lock()
			isCrawling = false
			crawlMutex.Unlock()
		}()

		result, _ := ensureCrawler().CrawlAllWithHistory()

		crawlMutex.Lock()
		lastCrawlResult = result
		crawlMutex.Unlock()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Batch crawling started (with history saving)",
		"status":  "Check /judal/status for progress",
	})
}

// GetStatus 크롤러 상태 조회
func (h *Handler) GetStatus(c *gin.Context) {
	repo := database.NewRepository()
	stats, _ := repo.GetStats()

	crawlMutex.Lock()
	status := gin.H{
		"is_crawling": isCrawling,
		"stats":       stats,
	}
	if lastCrawlResult != nil {
		status["last_crawl"] = lastCrawlResult
	}
	crawlMutex.Unlock()

	c.JSON(http.StatusOK, status)
}

// GetAvailableTabs 사용 가능한 실시간 탭 목록
func (h *Handler) GetAvailableTabs(c *gin.Context) {
	tabs := crawler.GetAvailableTabs()
	c.JSON(http.StatusOK, tabs)
}

// RealtimeThemeTab 테마 탭 실시간 크롤링
func (h *Handler) RealtimeThemeTab(c *gin.Context) {
	tab := c.Param("tab")

	rc := crawler.NewRealtimeCrawler()
	data, err := rc.CrawlThemeListTab(tab)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          err.Error(),
			"available_tabs": crawler.ThemeListURLs,
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// RealtimeStockTab 종목 탭 실시간 크롤링
func (h *Handler) RealtimeStockTab(c *gin.Context) {
	tab := c.Param("tab")

	rc := crawler.NewRealtimeCrawler()
	data, err := rc.CrawlStockListTab(tab)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          err.Error(),
			"available_tabs": crawler.StockListURLs,
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetStockHistory 종목 히스토리 조회
func (h *Handler) GetStockHistory(c *gin.Context) {
	code := c.Param("code")
	limit := 30
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	repo := database.NewRepository()
	history, err := repo.GetStockHistory(code, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    code,
		"count":   len(history),
		"history": history,
	})
}

// GetHistoryDates 히스토리 날짜 목록 조회
func (h *Handler) GetHistoryDates(c *gin.Context) {
	limit := 30
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	repo := database.NewRepository()
	dates, err := repo.GetHistoryDates(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(dates),
		"dates": dates,
	})
}

// GetCrawlLogs 크롤링 로그 조회
func (h *Handler) GetCrawlLogs(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	repo := database.NewRepository()
	logs, err := repo.GetCrawlLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(logs),
		"logs":  logs,
	})
}

// IngestThemes handles batch ingestion of themes
func (h *Handler) IngestThemes(c *gin.Context) {
	var themes []models.Theme
	if err := c.ShouldBindJSON(&themes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := database.NewRepository()
	count := 0
	for _, t := range themes {
		if err := repo.UpsertTheme(&t); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		count++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Themes ingested successfully",
		"count":   count,
	})
}

// IngestStocks handles batch ingestion of stocks
func (h *Handler) IngestStocks(c *gin.Context) {
	var stocks []models.Stock
	if err := c.ShouldBindJSON(&stocks); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := database.NewRepository()
	count := 0
	for _, s := range stocks {
		if err := repo.UpsertStock(&s); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		count++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stocks ingested successfully",
		"count":   count,
	})
}
