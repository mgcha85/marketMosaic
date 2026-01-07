package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dx-unified/internal/candle/model"

	"github.com/parquet-go/parquet-go"
)

// ParquetCandle represents a candle for Parquet serialization
type ParquetCandle struct {
	Symbol     string    `parquet:"symbol,dict"`
	Open       float64   `parquet:"open"`
	High       float64   `parquet:"high"`
	Low        float64   `parquet:"low"`
	Close      float64   `parquet:"close"`
	Volume     uint64    `parquet:"volume"`
	Timestamp  time.Time `parquet:"timestamp"`
	TradeCount uint64    `parquet:"trade_count"`
	VWAP       float64   `parquet:"vwap"`
}

// --- Instruments ---

func UpsertInstruments(instruments []model.Instrument) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, i := range instruments {
		_, err := tx.Exec(`
			INSERT INTO instruments (market, symbol, name, exchange, currency, market_cap, market_cap_ts, is_active, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(market, symbol) DO UPDATE SET
				name = excluded.name,
				exchange = excluded.exchange,
				currency = excluded.currency,
				market_cap = excluded.market_cap,
				market_cap_ts = excluded.market_cap_ts,
				is_active = excluded.is_active,
				updated_at = excluded.updated_at
		`, i.Market, i.Symbol, i.Name, i.Exchange, i.Currency,
			i.MarketCap, i.MarketCapTS, i.IsActive, i.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// --- Universe Snapshots ---

func SaveUniverseSnapshot(snap model.UniverseSnapshot) error {
	_, err := DB.Exec(`
		INSERT INTO universe_snapshots (ymd, market, market_cap_min, symbols_json, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(ymd, market, market_cap_min) DO UPDATE SET
			symbols_json = excluded.symbols_json,
			created_at = excluded.created_at
	`, snap.YMD, snap.Market, snap.MarketCapMin, snap.SymbolsJSON, snap.CreatedAt)
	return err
}

// --- Candles (Parquet) ---

// GetLastCandleTS gets the last timestamp for a symbol from Parquet files
func GetLastCandleTS(market, symbol, timeframe string) (int64, error) {
	pattern := GetParquetGlob(market, "", "")

	query := fmt.Sprintf(`
		SELECT COALESCE(MAX(epoch(timestamp)), 0) as max_ts
		FROM read_parquet('%s', hive_partitioning=true)
		WHERE symbol = ?
	`, pattern)

	var ts int64
	err := DB.QueryRow(query, symbol).Scan(&ts)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return ts, nil
}

// SaveCandlesToParquet saves candles to a Parquet file with Hive partitioning
// Structure: {DataDir}/market={market}/year=YYYY/month=MM/data_YYYYMMDD.parquet
func SaveCandlesToParquet(market string, date time.Time, candles []model.Candle) (int, error) {
	if len(candles) == 0 {
		return 0, nil
	}

	// Create directory with Hive partition structure
	dir := filepath.Join(
		DataDir,
		fmt.Sprintf("market=%s", market),
		fmt.Sprintf("year=%04d", date.Year()),
		fmt.Sprintf("month=%02d", date.Month()),
	)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create Parquet file
	filename := filepath.Join(dir, fmt.Sprintf("data_%s.parquet", date.Format("20060102")))

	f, err := os.Create(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer f.Close()

	// Convert to ParquetCandle
	pqCandles := make([]ParquetCandle, len(candles))
	for i, c := range candles {
		pqCandles[i] = ParquetCandle{
			Symbol:     c.Symbol,
			Open:       c.Open,
			High:       c.High,
			Low:        c.Low,
			Close:      c.Close,
			Volume:     uint64(c.Volume),
			Timestamp:  time.Unix(c.TS, 0),
			TradeCount: uint64(c.TradeCount),
			VWAP:       c.VWAP,
		}
	}

	// Write Parquet
	writer := parquet.NewGenericWriter[ParquetCandle](f)
	_, err = writer.Write(pqCandles)
	if err != nil {
		return 0, fmt.Errorf("failed to write parquet: %w", err)
	}

	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("failed to close parquet writer: %w", err)
	}

	return len(candles), nil
}

// UpsertCandles is deprecated - use SaveCandlesToParquet instead
// Kept for backward compatibility with service layer
func UpsertCandles(candles []model.Candle) (int, error) {
	if len(candles) == 0 {
		return 0, nil
	}

	// Group by market and date
	groups := make(map[string][]model.Candle)
	for _, c := range candles {
		ts := time.Unix(c.TS, 0)
		key := fmt.Sprintf("%s|%s", c.Market, ts.Format("20060102"))
		groups[key] = append(groups[key], c)
	}

	total := 0
	for key, groupCandles := range groups {
		// Parse key
		var market, dateStr string
		fmt.Sscanf(key, "%s|%s", &market, &dateStr)
		date, _ := time.Parse("20060102", dateStr)

		// Handle market parsing
		parts := filepath.SplitList(key)
		if len(parts) >= 1 {
			market = groupCandles[0].Market
			date = time.Unix(groupCandles[0].TS, 0)
		}

		count, err := SaveCandlesToParquet(market, date, groupCandles)
		if err != nil {
			return total, err
		}
		total += count
	}

	return total, nil
}

// --- Ingest Runs ---

func CreateIngestRun(run *model.IngestRun) error {
	err := DB.QueryRow(`
		INSERT INTO ingest_runs (started_at, finished_at, market, job, timeframe, symbols_count, inserted_rows, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, run.StartedAt, run.FinishedAt, run.Market, run.Job, run.Timeframe, run.SymbolsCount, run.InsertedRows, run.Status, run.ErrorMessage).Scan(&run.ID)
	return err
}

func UpdateIngestRun(run *model.IngestRun) error {
	_, err := DB.Exec(`
		UPDATE ingest_runs
		SET finished_at = ?, symbols_count = ?, inserted_rows = ?, status = ?, error_message = ?
		WHERE id = ?
	`, run.FinishedAt, run.SymbolsCount, run.InsertedRows, run.Status, run.ErrorMessage, run.ID)
	return err
}
