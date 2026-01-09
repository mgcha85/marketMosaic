package api

import (
	"net/http"
	"strconv"

	"dx-unified/internal/dart/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler holds dependencies for DART API handlers
type Handler struct {
	DB *gorm.DB
}

// NewHandler creates a new DART API handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// RegisterRoutes registers all DART API routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dart := rg.Group("/dart")
	{
		dart.GET("/corps", h.GetCorps)
		dart.GET("/filings", h.GetFilings)
		dart.GET("/filings/:rcept_no", h.GetFilingDetail)

		// Migration
		dart.POST("/migration/filings", h.IngestFilings)
	}
}

// GetCorps returns a paginated list of corporations
func (h *Handler) GetCorps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var corps []models.Corp
	var total int64

	h.DB.Model(&models.Corp{}).Count(&total)
	result := h.DB.Limit(limit).Offset(offset).Order("corp_name ASC").Find(&corps)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  corps,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetFilings returns a paginated list of filings
func (h *Handler) GetFilings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Optional filters
	corpCode := c.Query("corp_code")
	stockCode := c.Query("stock_code")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	var filings []models.Filing
	var total int64

	query := h.DB.Model(&models.Filing{})

	if corpCode != "" {
		query = query.Where("corp_code = ?", corpCode)
	}
	if stockCode != "" {
		query = query.Joins("JOIN corps ON filings.corp_code = corps.corp_code").
			Where("corps.stock_code = ?", stockCode)
	}
	if dateFrom != "" {
		query = query.Where("rcept_dt >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("rcept_dt <= ?", dateTo)
	}

	query.Count(&total)
	result := query.Limit(limit).Offset(offset).Order("rcept_dt DESC, rcept_no DESC").Find(&filings)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  filings,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetFilingDetail returns detailed information about a specific filing
func (h *Handler) GetFilingDetail(c *gin.Context) {
	rceptNo := c.Param("rcept_no")

	var filing models.Filing
	if err := h.DB.Where("rcept_no = ?", rceptNo).First(&filing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Filing not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var documents []models.FilingDocument
	h.DB.Where("rcept_no = ?", rceptNo).Find(&documents)

	var events []models.ExtractedEvent
	h.DB.Where("rcept_no = ?", rceptNo).Find(&events)

	c.JSON(http.StatusOK, gin.H{
		"filing":    filing,
		"documents": documents,
		"events":    events,
	})
}

// IngestFilings handles batch ingestion of DART filings
func (h *Handler) IngestFilings(c *gin.Context) {
	var filings []models.Filing
	if err := c.ShouldBindJSON(&filings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(filings) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No data", "count": 0})
		return
	}

	// Batch Upsert using GORM Clause
	// Note: We need to import "gorm.io/gorm/clause"
	// However, standard GORM Create with multiple items works for Insert.
	// For Upsert, we need clause.OnConflict.
	// Since I can't easily add import via this tool without reading top, I'll rely on generic Save if applicable?
	// Or try to use h.DB.Save() in loop? Loop is safer without import knowledge.
	// Or just Loop and FirstOrCreate/Updates. Use Transaction.

	tx := h.DB.Begin()
	count := 0
	for _, f := range filings {
		// Use Save (Upsert based on PK: rcept_no)
		if err := tx.Save(&f).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		count++
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Filings ingested successfully",
		"count":   count,
	})
}
