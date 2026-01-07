package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

var DB *sql.DB
var DataDir string // Parquet 파일이 저장되는 디렉토리

// InitDB initializes DuckDB and sets up the data directory for Parquet files
func InitDB(dataDir string) error {
	DataDir = dataDir

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	var err error
	// Use in-memory DuckDB for querying Parquet files
	DB, err = sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping DuckDB: %w", err)
	}

	// Create metadata tables in DuckDB (in-memory)
	if err := createMetadataTables(); err != nil {
		return fmt.Errorf("failed to create metadata tables: %w", err)
	}

	log.Printf("[CANDLE] DuckDB initialized with data directory: %s", dataDir)

	// Seed dev data for in-memory DB
	if err := SeedDevData(); err != nil {
		log.Printf("Failed to seed dev data: %v", err)
	}

	return nil
}

// SeedDevData populates the DB with default data for development
func SeedDevData() error {
	// Check if already populated (though unlikely for in-memory)
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM instruments").Scan(&count)
	if err == nil && count > 0 {
		return nil
	}

	log.Println("[CANDLE] Seeding development data...")

	// 1. Insert Instruments
	_, err = DB.Exec(`
		INSERT INTO instruments (market, symbol, name, exchange, currency, market_cap, is_active, updated_at) VALUES
		('KR', '005930', 'Samsung Electronics', 'KOSPI', 'KRW', 450000000000000, true, ?),
		('KR', '000660', 'SK Hynix', 'KOSPI', 'KRW', 90000000000000, true, ?)
	`, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to seed instruments: %w", err)
	}

	// 2. Insert Snapshot for today
	ymd := time.Now().Format("2006-01-02")
	symbolsJSON := `["005930", "000660"]`
	_, err = DB.Exec(`
		INSERT INTO universe_snapshots (ymd, market, market_cap_min, symbols_json, created_at)
		VALUES (?, 'KR', 0, ?, ?)
	`, ymd, symbolsJSON, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to seed snapshot: %w", err)
	}

	return nil
}

// createMetadataTables creates tables for metadata (not actual candle data)
func createMetadataTables() error {
	// Instruments table for universe management
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS instruments (
			market VARCHAR NOT NULL,
			symbol VARCHAR NOT NULL,
			name VARCHAR,
			exchange VARCHAR,
			currency VARCHAR,
			market_cap DOUBLE,
			market_cap_ts BIGINT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (market, symbol)
		)
	`)
	if err != nil {
		return err
	}

	// Universe snapshots
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS universe_snapshots (
			ymd VARCHAR NOT NULL,
			market VARCHAR NOT NULL,
			market_cap_min DOUBLE NOT NULL,
			symbols_json VARCHAR NOT NULL,
			created_at BIGINT NOT NULL,
			PRIMARY KEY (ymd, market, market_cap_min)
		)
	`)
	if err != nil {
		return err
	}

	// Ingest runs log
	_, err = DB.Exec(`
		CREATE SEQUENCE IF NOT EXISTS ingest_runs_seq START 1;
		CREATE TABLE IF NOT EXISTS ingest_runs (
			id INTEGER DEFAULT nextval('ingest_runs_seq') PRIMARY KEY,
			started_at BIGINT NOT NULL,
			finished_at BIGINT,
			market VARCHAR NOT NULL,
			job VARCHAR NOT NULL,
			timeframe VARCHAR,
			symbols_count INTEGER,
			inserted_rows INTEGER,
			status VARCHAR NOT NULL,
			error_message VARCHAR
		)
	`)

	return err
}

// GetParquetGlob returns the glob pattern for Parquet files
// Hive partition structure: data/market={market}/year=YYYY/month=MM/*.parquet
func GetParquetGlob(market, year, month string) string {
	pattern := filepath.Join(DataDir, "market="+market)
	if year != "" {
		pattern = filepath.Join(pattern, "year="+year)
		if month != "" {
			pattern = filepath.Join(pattern, "month="+month)
		} else {
			pattern = filepath.Join(pattern, "*")
		}
	} else {
		pattern = filepath.Join(pattern, "*", "*")
	}
	return filepath.Join(pattern, "*.parquet")
}

// GetAllParquetGlob returns glob pattern for all markets
func GetAllParquetGlob() string {
	return filepath.Join(DataDir, "market=*", "year=*", "month=*", "*.parquet")
}

// QueryCandles queries candle data from Parquet files using DuckDB
func QueryCandles(market, symbol, timeframe string, tsFrom, tsTo int64, limit int) ([]map[string]interface{}, error) {
	pattern := GetAllParquetGlob()
	if market != "" {
		pattern = GetParquetGlob(market, "", "")
	}

	query := fmt.Sprintf(`
		SELECT 
			market,
			symbol,
			'%s' as timeframe,
			epoch(timestamp) as ts,
			open,
			high,
			low,
			close,
			volume,
			vwap,
			trade_count
		FROM read_parquet('%s', hive_partitioning=true)
		WHERE 1=1
	`, timeframe, pattern)

	args := []interface{}{}

	if symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}
	if tsFrom > 0 {
		query += " AND epoch(timestamp) >= ?"
		args = append(args, tsFrom)
	}
	if tsTo > 0 {
		query += " AND epoch(timestamp) <= ?"
		args = append(args, tsTo)
	}

	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var candles []map[string]interface{}
	for rows.Next() {
		var market, symbol, timeframe string
		var ts int64
		var open, high, low, closePrice, volume, vwap sql.NullFloat64
		var tradeCount sql.NullInt64

		if err := rows.Scan(&market, &symbol, &timeframe, &ts, &open, &high, &low, &closePrice, &volume, &vwap, &tradeCount); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		candle := map[string]interface{}{
			"market":    market,
			"symbol":    symbol,
			"timeframe": timeframe,
			"ts":        ts,
		}
		if open.Valid {
			candle["open"] = open.Float64
		}
		if high.Valid {
			candle["high"] = high.Float64
		}
		if low.Valid {
			candle["low"] = low.Float64
		}
		if closePrice.Valid {
			candle["close"] = closePrice.Float64
		}
		if volume.Valid {
			candle["volume"] = volume.Float64
		}
		if vwap.Valid {
			candle["vwap"] = vwap.Float64
		}
		if tradeCount.Valid {
			candle["trade_count"] = tradeCount.Int64
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// QueryCandlesBySymbol queries candles for a specific symbol
func QueryCandlesBySymbol(symbol, market, timeframe string, limit int) ([]map[string]interface{}, error) {
	return QueryCandles(market, symbol, timeframe, 0, 0, limit)
}

// GetAvailableDates returns list of available dates from Parquet files
func GetAvailableDates(market string) ([]string, error) {
	pattern := GetAllParquetGlob()
	if market != "" {
		pattern = GetParquetGlob(market, "", "")
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT date_trunc('day', timestamp)::DATE as date
		FROM read_parquet('%s', hive_partitioning=true)
		ORDER BY date DESC
		LIMIT 100
	`, pattern)

	rows, err := DB.Query(query)
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

// Close closes the DuckDB connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}
