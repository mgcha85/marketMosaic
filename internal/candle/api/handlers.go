package api

import (
	"database/sql"
	"net/http"
	"strconv"

	candleDB "dx-unified/internal/candle/database"

	"github.com/gin-gonic/gin"
)

// Handler holds dependencies for Candle API handlers
type Handler struct{}

// NewHandler creates a new Candle API handler
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers all Candle API routes under /candle prefix
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	candle := rg.Group("/candle")
	{
		// Universe
		candle.GET("/universe", h.GetUniverse)

		// Candles (from Parquet files via DuckDB)
		candle.GET("/stocks", h.GetCandles)
		candle.GET("/stocks/:symbol", h.GetCandlesBySymbol)

		// Available dates
		candle.GET("/dates", h.GetAvailableDates)

		// Runs
		candle.GET("/runs", h.GetRuns)
	}
}

// GetUniverse returns universe (active instruments)
func (h *Handler) GetUniverse(c *gin.Context) {
	market := c.DefaultQuery("market", "")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	query := `
		SELECT market, symbol, name, exchange, currency, market_cap, is_active, updated_at
		FROM instruments
		WHERE is_active = true
	`
	args := []interface{}{}

	if market != "" {
		query += " AND market = ?"
		args = append(args, market)
	}

	query += " ORDER BY market_cap DESC NULLS LAST LIMIT ?"
	args = append(args, limit)

	rows, err := candleDB.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var instruments []map[string]interface{}
	for rows.Next() {
		var market, symbol string
		var name, exchange, currency sql.NullString
		var marketCap sql.NullFloat64
		var isActive bool
		var updatedAt int64

		if err := rows.Scan(&market, &symbol, &name, &exchange, &currency, &marketCap, &isActive, &updatedAt); err != nil {
			continue
		}

		inst := map[string]interface{}{
			"market":     market,
			"symbol":     symbol,
			"is_active":  isActive,
			"updated_at": updatedAt,
		}
		if name.Valid {
			inst["name"] = name.String
		}
		if exchange.Valid {
			inst["exchange"] = exchange.String
		}
		if currency.Valid {
			inst["currency"] = currency.String
		}
		if marketCap.Valid {
			inst["market_cap"] = marketCap.Float64
		}
		instruments = append(instruments, inst)
	}

	c.JSON(http.StatusOK, gin.H{
		"count":       len(instruments),
		"instruments": instruments,
	})
}

// GetCandles returns candle data from Parquet files using DuckDB
func (h *Handler) GetCandles(c *gin.Context) {
	market := c.Query("market")
	symbol := c.Query("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	var tsFrom, tsTo int64
	if dateFrom != "" {
		tsFrom, _ = strconv.ParseInt(dateFrom, 10, 64)
	}
	if dateTo != "" {
		tsTo, _ = strconv.ParseInt(dateTo, 10, 64)
	}

	candles, err := candleDB.QueryCandles(market, symbol, timeframe, tsFrom, tsTo, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "Failed to query Parquet files. Make sure data exists in the correct Hive partition structure.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(candles),
		"timeframe": timeframe,
		"candles":   candles,
	})
}

// GetCandlesBySymbol returns candles for specific symbol
func (h *Handler) GetCandlesBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	market := c.DefaultQuery("market", "")
	timeframe := c.DefaultQuery("timeframe", "1m")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	candles, err := candleDB.QueryCandlesBySymbol(symbol, market, timeframe, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "Failed to query Parquet files",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timeframe": timeframe,
		"count":     len(candles),
		"candles":   candles,
	})
}

// GetAvailableDates returns list of available dates in the Parquet files
func (h *Handler) GetAvailableDates(c *gin.Context) {
	market := c.Query("market")

	dates, err := candleDB.GetAvailableDates(market)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(dates),
		"dates": dates,
	})
}

// GetRuns returns ingest run logs
func (h *Handler) GetRuns(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	rows, err := candleDB.DB.Query(`
		SELECT id, started_at, finished_at, market, job, timeframe, symbols_count, inserted_rows, status, error_message
		FROM ingest_runs
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var runs []map[string]interface{}
	for rows.Next() {
		var id int64
		var startedAt, finishedAt, symbolsCount, insertedRows sql.NullInt64
		var market, job, timeframe, status, errorMsg sql.NullString

		if err := rows.Scan(&id, &startedAt, &finishedAt, &market, &job, &timeframe, &symbolsCount, &insertedRows, &status, &errorMsg); err != nil {
			continue
		}

		run := map[string]interface{}{
			"id": id,
		}
		if startedAt.Valid {
			run["started_at"] = startedAt.Int64
		}
		if finishedAt.Valid {
			run["finished_at"] = finishedAt.Int64
		}
		if market.Valid {
			run["market"] = market.String
		}
		if job.Valid {
			run["job"] = job.String
		}
		if timeframe.Valid {
			run["timeframe"] = timeframe.String
		}
		if symbolsCount.Valid {
			run["symbols_count"] = symbolsCount.Int64
		}
		if insertedRows.Valid {
			run["inserted_rows"] = insertedRows.Int64
		}
		if status.Valid {
			run["status"] = status.String
		}
		if errorMsg.Valid && errorMsg.String != "" {
			run["error_message"] = errorMsg.String
		}

		runs = append(runs, run)
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(runs),
		"runs":  runs,
	})
}
