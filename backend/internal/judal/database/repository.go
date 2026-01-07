package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"dx-unified/internal/judal/models"
)

// Repository 데이터베이스 작업을 위한 인터페이스
type Repository struct {
	db *sql.DB
}

// NewRepository 새 레포지토리 생성
func NewRepository() *Repository {
	if DB == nil {
		log.Println("Warning: Database not initialized, returning repository with nil db")
	}
	return &Repository{db: DB}
}

// GetDB 현재 데이터베이스 연결 반환
func GetDB() *sql.DB {
	return DB
}

// ===== Theme Operations =====

// UpsertTheme 테마 추가 또는 업데이트
func (r *Repository) UpsertTheme(theme *models.Theme) error {
	query := `
		INSERT INTO themes (theme_idx, name, stock_count, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(theme_idx) DO UPDATE SET
			name = excluded.name,
			stock_count = excluded.stock_count,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, theme.ThemeIdx, theme.Name, theme.StockCount)
	return err
}

// GetAllThemes 모든 테마 조회
func (r *Repository) GetAllThemes() ([]models.Theme, error) {
	query := `SELECT id, theme_idx, name, stock_count, created_at, updated_at FROM themes ORDER BY name`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []models.Theme
	for rows.Next() {
		var t models.Theme
		if err := rows.Scan(&t.ID, &t.ThemeIdx, &t.Name, &t.StockCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			log.Printf("Error scanning theme: %v", err)
			continue
		}
		themes = append(themes, t)
	}
	return themes, rows.Err()
}

// GetThemeByIdx themeIdx로 테마 조회
func (r *Repository) GetThemeByIdx(themeIdx int) (*models.Theme, error) {
	query := `SELECT id, theme_idx, name, stock_count, created_at, updated_at FROM themes WHERE theme_idx = ?`
	var t models.Theme
	err := r.db.QueryRow(query, themeIdx).Scan(&t.ID, &t.ThemeIdx, &t.Name, &t.StockCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ===== Stock Operations =====

// UpsertStock 종목 추가 또는 업데이트
func (r *Repository) UpsertStock(stock *models.Stock) error {
	query := `
		INSERT INTO stocks (
			code, name, market, current_price, price_change, change_rate,
			three_day_sum, high_52w, low_52w, change_rate_52w_up, change_rate_52w_down,
			neglect_index_52w, high_3y, low_3y, change_rate_3y_up, change_rate_3y_down,
			neglect_index_3y, price_index_3y, expected_return, pbr, per, eps,
			market_cap, volume_index, volume_index_7d, buffett_choice, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			market = excluded.market,
			current_price = excluded.current_price,
			price_change = excluded.price_change,
			change_rate = excluded.change_rate,
			three_day_sum = excluded.three_day_sum,
			high_52w = excluded.high_52w,
			low_52w = excluded.low_52w,
			change_rate_52w_up = excluded.change_rate_52w_up,
			change_rate_52w_down = excluded.change_rate_52w_down,
			neglect_index_52w = excluded.neglect_index_52w,
			high_3y = excluded.high_3y,
			low_3y = excluded.low_3y,
			change_rate_3y_up = excluded.change_rate_3y_up,
			change_rate_3y_down = excluded.change_rate_3y_down,
			neglect_index_3y = excluded.neglect_index_3y,
			price_index_3y = excluded.price_index_3y,
			expected_return = excluded.expected_return,
			pbr = excluded.pbr,
			per = excluded.per,
			eps = excluded.eps,
			market_cap = excluded.market_cap,
			volume_index = excluded.volume_index,
			volume_index_7d = excluded.volume_index_7d,
			buffett_choice = excluded.buffett_choice,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query,
		stock.Code, stock.Name, stock.Market,
		stock.CurrentPrice, stock.PriceChange, stock.ChangeRate,
		stock.ThreeDaySum, stock.High52W, stock.Low52W,
		stock.ChangeRate52WUp, stock.ChangeRate52WDown, stock.NeglectIndex52W,
		stock.High3Y, stock.Low3Y, stock.ChangeRate3YUp, stock.ChangeRate3YDown,
		stock.NeglectIndex3Y, stock.PriceIndex3Y, stock.ExpectedReturn,
		stock.PBR, stock.PER, stock.EPS, stock.MarketCap,
		stock.VolumeIndex, stock.VolumeIndex7D, stock.BuffettChoice,
	)
	return err
}

// GetStockByCode 종목코드로 종목 조회
func (r *Repository) GetStockByCode(code string) (*models.Stock, error) {
	query := `
		SELECT id, code, name, market, current_price, price_change, change_rate,
			three_day_sum, high_52w, low_52w, change_rate_52w_up, change_rate_52w_down,
			neglect_index_52w, high_3y, low_3y, change_rate_3y_up, change_rate_3y_down,
			neglect_index_3y, price_index_3y, expected_return, pbr, per, eps,
			market_cap, volume_index, volume_index_7d, buffett_choice, created_at, updated_at
		FROM stocks WHERE code = ?
	`
	var s models.Stock
	err := r.db.QueryRow(query, code).Scan(
		&s.ID, &s.Code, &s.Name, &s.Market,
		&s.CurrentPrice, &s.PriceChange, &s.ChangeRate,
		&s.ThreeDaySum, &s.High52W, &s.Low52W,
		&s.ChangeRate52WUp, &s.ChangeRate52WDown, &s.NeglectIndex52W,
		&s.High3Y, &s.Low3Y, &s.ChangeRate3YUp, &s.ChangeRate3YDown,
		&s.NeglectIndex3Y, &s.PriceIndex3Y, &s.ExpectedReturn,
		&s.PBR, &s.PER, &s.EPS, &s.MarketCap,
		&s.VolumeIndex, &s.VolumeIndex7D, &s.BuffettChoice,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// StockQueryParams 종목 조회 파라미터
type StockQueryParams struct {
	Sort   string
	Order  string
	Limit  int
	Offset int
	Market string
}

// GetStocks 종목 목록 조회 (필터링, 정렬 지원)
func (r *Repository) GetStocks(params StockQueryParams) ([]models.Stock, int, error) {
	// 허용된 정렬 필드
	allowedSortFields := map[string]bool{
		"name": true, "code": true, "current_price": true, "price_change": true,
		"change_rate": true, "three_day_sum": true, "high_52w": true, "low_52w": true,
		"change_rate_52w_up": true, "change_rate_52w_down": true, "neglect_index_52w": true,
		"price_index_3y": true, "expected_return": true, "pbr": true, "per": true,
		"eps": true, "market_cap": true, "volume_index": true, "buffett_choice": true,
		"updated_at": true,
	}

	// 기본값 설정
	sortField := "name"
	if params.Sort != "" && allowedSortFields[params.Sort] {
		sortField = params.Sort
	}

	order := "ASC"
	if strings.ToUpper(params.Order) == "DESC" {
		order = "DESC"
	}

	limit := 100
	if params.Limit > 0 && params.Limit <= 500 {
		limit = params.Limit
	}

	offset := 0
	if params.Offset > 0 {
		offset = params.Offset
	}

	// WHERE 절 구성
	var whereConditions []string
	var args []interface{}

	if params.Market != "" {
		whereConditions = append(whereConditions, "market = ?")
		args = append(args, params.Market)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 총 개수 조회
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stocks %s", whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 데이터 조회
	query := fmt.Sprintf(`
		SELECT id, code, name, market, current_price, price_change, change_rate,
			three_day_sum, high_52w, low_52w, change_rate_52w_up, change_rate_52w_down,
			neglect_index_52w, high_3y, low_3y, change_rate_3y_up, change_rate_3y_down,
			neglect_index_3y, price_index_3y, expected_return, pbr, per, eps,
			market_cap, volume_index, volume_index_7d, buffett_choice, created_at, updated_at
		FROM stocks %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortField, order)

	args = append(args, limit, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var stocks []models.Stock
	for rows.Next() {
		var s models.Stock
		if err := rows.Scan(
			&s.ID, &s.Code, &s.Name, &s.Market,
			&s.CurrentPrice, &s.PriceChange, &s.ChangeRate,
			&s.ThreeDaySum, &s.High52W, &s.Low52W,
			&s.ChangeRate52WUp, &s.ChangeRate52WDown, &s.NeglectIndex52W,
			&s.High3Y, &s.Low3Y, &s.ChangeRate3YUp, &s.ChangeRate3YDown,
			&s.NeglectIndex3Y, &s.PriceIndex3Y, &s.ExpectedReturn,
			&s.PBR, &s.PER, &s.EPS, &s.MarketCap,
			&s.VolumeIndex, &s.VolumeIndex7D, &s.BuffettChoice,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning stock: %v", err)
			continue
		}
		stocks = append(stocks, s)
	}

	return stocks, total, rows.Err()
}

// ===== Theme-Stock Mapping Operations =====

// AddThemeStock 테마-종목 매핑 추가
func (r *Repository) AddThemeStock(themeIdx int, stockCode string) error {
	query := `INSERT OR IGNORE INTO theme_stocks (theme_idx, stock_code) VALUES (?, ?)`
	_, err := r.db.Exec(query, themeIdx, stockCode)
	return err
}

// GetStocksByTheme 테마별 종목 조회
func (r *Repository) GetStocksByTheme(themeIdx int) ([]models.Stock, error) {
	query := `
		SELECT s.id, s.code, s.name, s.market, s.current_price, s.price_change, s.change_rate,
			s.three_day_sum, s.high_52w, s.low_52w, s.change_rate_52w_up, s.change_rate_52w_down,
			s.neglect_index_52w, s.high_3y, s.low_3y, s.change_rate_3y_up, s.change_rate_3y_down,
			s.neglect_index_3y, s.price_index_3y, s.expected_return, s.pbr, s.per, s.eps,
			s.market_cap, s.volume_index, s.volume_index_7d, s.buffett_choice, s.created_at, s.updated_at
		FROM stocks s
		JOIN theme_stocks ts ON s.code = ts.stock_code
		WHERE ts.theme_idx = ?
		ORDER BY s.name
	`
	rows, err := r.db.Query(query, themeIdx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []models.Stock
	for rows.Next() {
		var s models.Stock
		if err := rows.Scan(
			&s.ID, &s.Code, &s.Name, &s.Market,
			&s.CurrentPrice, &s.PriceChange, &s.ChangeRate,
			&s.ThreeDaySum, &s.High52W, &s.Low52W,
			&s.ChangeRate52WUp, &s.ChangeRate52WDown, &s.NeglectIndex52W,
			&s.High3Y, &s.Low3Y, &s.ChangeRate3YUp, &s.ChangeRate3YDown,
			&s.NeglectIndex3Y, &s.PriceIndex3Y, &s.ExpectedReturn,
			&s.PBR, &s.PER, &s.EPS, &s.MarketCap,
			&s.VolumeIndex, &s.VolumeIndex7D, &s.BuffettChoice,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning stock: %v", err)
			continue
		}
		stocks = append(stocks, s)
	}

	return stocks, rows.Err()
}

// GetThemesByStock 종목이 속한 테마 목록 조회
func (r *Repository) GetThemesByStock(stockCode string) ([]models.Theme, error) {
	query := `
		SELECT t.id, t.theme_idx, t.name, t.stock_count, t.created_at, t.updated_at
		FROM themes t
		JOIN theme_stocks ts ON t.theme_idx = ts.theme_idx
		WHERE ts.stock_code = ?
		ORDER BY t.name
	`
	rows, err := r.db.Query(query, stockCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []models.Theme
	for rows.Next() {
		var t models.Theme
		if err := rows.Scan(&t.ID, &t.ThemeIdx, &t.Name, &t.StockCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		themes = append(themes, t)
	}
	return themes, rows.Err()
}

// ClearThemeStocks 테마-종목 매핑 초기화 (해당 테마)
func (r *Repository) ClearThemeStocks(themeIdx int) error {
	_, err := r.db.Exec("DELETE FROM theme_stocks WHERE theme_idx = ?", themeIdx)
	return err
}

// ===== Statistics =====

// GetStats 통계 정보
type Stats struct {
	ThemeCount    int       `json:"theme_count"`
	StockCount    int       `json:"stock_count"`
	MappingCount  int       `json:"mapping_count"`
	HistoryCount  int       `json:"history_count"`
	LastUpdated   time.Time `json:"last_updated"`
	LastCrawlDate string    `json:"last_crawl_date"`
}

// GetStats 통계 조회
func (r *Repository) GetStats() (*Stats, error) {
	stats := &Stats{}

	r.db.QueryRow("SELECT COUNT(*) FROM themes").Scan(&stats.ThemeCount)
	r.db.QueryRow("SELECT COUNT(*) FROM stocks").Scan(&stats.StockCount)
	r.db.QueryRow("SELECT COUNT(*) FROM theme_stocks").Scan(&stats.MappingCount)
	r.db.QueryRow("SELECT COUNT(*) FROM stock_history").Scan(&stats.HistoryCount)
	r.db.QueryRow("SELECT MAX(updated_at) FROM stocks").Scan(&stats.LastUpdated)
	r.db.QueryRow("SELECT MAX(crawl_date) FROM stock_history").Scan(&stats.LastCrawlDate)

	return stats, nil
}

// ===== History Operations =====

// SaveStockHistory 종목 히스토리 저장 (일별 스냅샷)
func (r *Repository) SaveStockHistory(crawlDate string, stock *models.Stock) error {
	query := `
		INSERT INTO stock_history (
			crawl_date, code, name, market, current_price, price_change, change_rate,
			three_day_sum, high_52w, low_52w, neglect_index_52w, price_index_3y,
			expected_return, pbr, per, eps, market_cap, volume_index
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(crawl_date, code) DO UPDATE SET
			name = excluded.name,
			market = excluded.market,
			current_price = excluded.current_price,
			price_change = excluded.price_change,
			change_rate = excluded.change_rate,
			three_day_sum = excluded.three_day_sum,
			high_52w = excluded.high_52w,
			low_52w = excluded.low_52w,
			neglect_index_52w = excluded.neglect_index_52w,
			price_index_3y = excluded.price_index_3y,
			expected_return = excluded.expected_return,
			pbr = excluded.pbr,
			per = excluded.per,
			eps = excluded.eps,
			market_cap = excluded.market_cap,
			volume_index = excluded.volume_index
	`
	_, err := r.db.Exec(query,
		crawlDate, stock.Code, stock.Name, stock.Market,
		stock.CurrentPrice, stock.PriceChange, stock.ChangeRate,
		stock.ThreeDaySum, stock.High52W, stock.Low52W,
		stock.NeglectIndex52W, stock.PriceIndex3Y, stock.ExpectedReturn,
		stock.PBR, stock.PER, stock.EPS, stock.MarketCap, stock.VolumeIndex,
	)
	return err
}

// GetStockHistory 특정 종목의 히스토리 조회
func (r *Repository) GetStockHistory(code string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT crawl_date, code, name, market, current_price, price_change, change_rate,
			high_52w, low_52w, neglect_index_52w, price_index_3y, expected_return,
			pbr, per, eps, market_cap
		FROM stock_history
		WHERE code = ?
		ORDER BY crawl_date DESC
		LIMIT ?
	`
	rows, err := r.db.Query(query, code, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var crawlDate, codeVal, name, market sql.NullString
		var currentPrice, priceChange, high52W, low52W, eps, marketCap sql.NullInt64
		var changeRate, neglectIndex52W, priceIndex3Y, expectedReturn, pbr, per sql.NullFloat64

		if err := rows.Scan(
			&crawlDate, &codeVal, &name, &market, &currentPrice, &priceChange, &changeRate,
			&high52W, &low52W, &neglectIndex52W, &priceIndex3Y, &expectedReturn,
			&pbr, &per, &eps, &marketCap,
		); err != nil {
			continue
		}

		record := map[string]interface{}{
			"crawl_date": crawlDate.String,
			"code":       codeVal.String,
			"name":       name.String,
			"market":     market.String,
		}
		if currentPrice.Valid {
			record["current_price"] = currentPrice.Int64
		}
		if changeRate.Valid {
			record["change_rate"] = changeRate.Float64
		}
		if pbr.Valid {
			record["pbr"] = pbr.Float64
		}
		if per.Valid {
			record["per"] = per.Float64
		}
		if marketCap.Valid {
			record["market_cap"] = marketCap.Int64
		}

		history = append(history, record)
	}

	return history, nil
}

// GetHistoryDates 히스토리에 있는 날짜 목록 조회
func (r *Repository) GetHistoryDates(limit int) ([]string, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `SELECT DISTINCT crawl_date FROM stock_history ORDER BY crawl_date DESC LIMIT ?`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			continue
		}
		dates = append(dates, date)
	}
	return dates, nil
}

// ===== Crawl Log Operations =====

// SaveCrawlLog 크롤링 로그 저장
func (r *Repository) SaveCrawlLog(crawlDate, crawlType string, themesCount, stocksCount int, durationSec float64, status string) error {
	query := `
		INSERT INTO crawl_logs (crawl_date, crawl_type, themes_count, stocks_count, duration_seconds, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, crawlDate, crawlType, themesCount, stocksCount, durationSec, status)
	return err
}

// GetCrawlLogs 크롤링 로그 조회
func (r *Repository) GetCrawlLogs(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT id, crawl_date, crawl_type, themes_count, stocks_count, duration_seconds, status, created_at
		FROM crawl_logs
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var crawlDate, crawlType, status, createdAt string
		var themesCount, stocksCount int
		var durationSec float64

		if err := rows.Scan(&id, &crawlDate, &crawlType, &themesCount, &stocksCount, &durationSec, &status, &createdAt); err != nil {
			continue
		}

		logs = append(logs, map[string]interface{}{
			"id":               id,
			"crawl_date":       crawlDate,
			"crawl_type":       crawlType,
			"themes_count":     themesCount,
			"stocks_count":     stocksCount,
			"duration_seconds": durationSec,
			"status":           status,
			"created_at":       createdAt,
		})
	}

	return logs, nil
}
