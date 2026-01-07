package candles

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	database "dx-unified/internal/candle/database"
	db "dx-unified/internal/candle/database"
	"dx-unified/internal/candle/model"
	"dx-unified/internal/candle/service"
)

type Service struct {
	SingleProvider service.CandleProvider      // Kiwoom
	MultiProvider  service.MultiCandleProvider // Alpaca
}

func NewService(single service.CandleProvider, multi service.MultiCandleProvider) *Service {
	return &Service{
		SingleProvider: single,
		MultiProvider:  multi,
	}
}

// IngestParams defines parameters for ingestion run.
type IngestParams struct {
	Market    string
	Timeframe string
	YMD       string // Target universe date
}

// Run executes the ingestion process.
func (s *Service) Run(params IngestParams) error {
	// 1. Start Log
	run := &model.IngestRun{
		StartedAt: time.Now().Unix(),
		Market:    params.Market,
		Job:       "candles",
		Timeframe: params.Timeframe,
		Status:    "running",
	}
	if err := database.CreateIngestRun(run); err != nil {
		return fmt.Errorf("failed to create run log: %w", err)
	}

	// 2. Load Universe
	// We need to get the symbols from 'universe_snapshots'.
	// This requires a DB query we haven't strictly written yet in 'db/queries.go',
	// or we can just query 'instruments' if we want "current universe".
	// The requirement: "today 수집 대상 종목(universe)".
	// Let's assume we query universe_snapshots for the YMD.
	// If YMD is not provided, use today.
	if params.YMD == "" {
		params.YMD = time.Now().Format("2006-01-02")
	}

	symbols, err := s.getSymbolsFromSnapshot(params.YMD, params.Market)
	if err != nil {
		return s.failRun(run, fmt.Sprintf("failed to load universe: %v", err))
	}

	run.SymbolsCount = int64(len(symbols))
	if err := db.UpdateIngestRun(run); err != nil {
		log.Printf("failed to update run: %v", err)
	}

	var totalInserted int64
	var ingestErr error

	// 3. Fetch & Insert Strategy
	if params.Market == model.MarketUS && s.MultiProvider != nil {
		totalInserted, ingestErr = s.ingestUS(symbols, params.Timeframe)
	} else if params.Market == model.MarketKR && s.SingleProvider != nil {
		totalInserted, ingestErr = s.ingestKR(symbols, params.Timeframe)
	} else {
		ingestErr = fmt.Errorf("no suitable provider for market %s", params.Market)
	}

	// 4. Finish Log
	run.FinishedAt = time.Now().Unix()
	run.InsertedRows = totalInserted
	if ingestErr != nil {
		run.Status = "failed"
		run.ErrorMessage = ingestErr.Error()
	} else {
		run.Status = "success"
	}
	db.UpdateIngestRun(run)

	return ingestErr
}

func (s *Service) getSymbolsFromSnapshot(ymd, market string) ([]string, error) {
	// Retrieve from DB.
	// We implement the query inline here or add to db/queries.go.
	// For speed, let's use global DB object here if permissible, or better add query.
	// I will add a raw query here for simplicity since I can't easily jump back to db package without context switch cost.

	var symbolsJSON string
	// LIMIT 1 because PK includes market_cap_min which we don't know here.
	// Ideally we select the generic or largest one?
	// Requirement says "today collection target".
	// We'll select the most recent created one for that YMD/Market.
	err := db.DB.QueryRow(`
        SELECT symbols_json FROM universe_snapshots 
        WHERE ymd = ? AND market = ? 
        ORDER BY created_at DESC LIMIT 1
    `, ymd, market).Scan(&symbolsJSON)

	if err != nil {
		return nil, err
	}

	var symbols []string
	if err := json.Unmarshal([]byte(symbolsJSON), &symbols); err != nil {
		return nil, err
	}
	return symbols, nil
}

func (s *Service) ingestUS(symbols []string, timeframe string) (int64, error) {
	// Optimization: Group symbols by LastTS to efficient bulk requests?
	// Alpaca allows 'start' param.
	// If symbols have different LastTS, we might need multiple batches or use the minimum LastTS and filter duplicates.
	// "Filter duplicates" is handled by UPDATE OR IGNORE in DB, so fetching overlapping data is safe but wasteful.
	// To minimize waste:
	// 1. Get LastTS for all symbols.
	// 2. Group by approximate LastTS (e.g. daily buckets) or just use logic "Min(LastTS)" for the chunk.
	// Since Alpaca Free tier is slow, let's keep it simple: chunk by 100, use min LastTS of the chunk.

	var totalInserted int64
	chunkSize := 100 // Alpaca limit

	for i := 0; i < len(symbols); i += chunkSize {
		end := i + chunkSize
		if end > len(symbols) {
			end = len(symbols)
		}
		chunk := symbols[i:end]

		// Find min LastTS in this chunk to decide 'start'
		minTS := int64(9999999999)
		for _, sym := range chunk {
			ts, err := db.GetLastCandleTS(model.MarketUS, sym, timeframe)
			if err != nil {
				// log error? continue
				ts = 0
			}
			if ts < minTS {
				minTS = ts
			}
		}

		// Alpaca 'start' param expects RFC3339
		// If minTS == 0, maybe define a default start (e.g. 1 year ago? or user prompt: "recent 30 days/90 days")
		// Prompt: "initial backfill (e.g. recent 30 days...)"

		var startTime string
		if minTS == 0 {
			// Default backfill: 30 days
			startTime = time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
		} else {
			// last_ts + 1 second?
			t := time.Unix(minTS+1, 0)
			startTime = t.Format(time.RFC3339)
		}

		barsMap, err := s.MultiProvider.FetchMultiBars(chunk, timeframe, startTime, "") // end="" means now
		if err != nil {
			log.Printf("failed to fetch bars for chunk %v: %v", chunk[0], err)
			continue
		}

		// Insert
		for _, bars := range barsMap {
			n, err := db.UpsertCandles(bars)
			if err != nil {
				log.Printf("failed to upsert candles for %s: %v", bars[0].Symbol, err)
			}
			totalInserted += int64(n)
		}
	}

	return totalInserted, nil
}

func (s *Service) ingestKR(symbols []string, timeframe string) (int64, error) {
	var totalInserted int64

	for _, sym := range symbols {
		lastTS, err := db.GetLastCandleTS(model.MarketKR, sym, timeframe)
		if err != nil {
			lastTS = 0
		}

		// Kiwoom provider handles logic internally? No, provider just fetches.
		// Our Kiwoom candles.go skeleton needs 'lastTS'??
		// I defined FetchCandles(symbol, timeframe, lastTS) in interface/implementation.
		// So we pass it down.

		candles, err := s.SingleProvider.FetchCandles(sym, timeframe, lastTS)
		if err != nil {
			log.Printf("failed to fetch KR candles for %s: %v", sym, err)
			continue
		}

		// Filter candles <= lastTS just in case provider returns overlap
		var newCandles []model.Candle
		for _, c := range candles {
			if c.TS > lastTS {
				newCandles = append(newCandles, c)
			}
		}

		n, err := db.UpsertCandles(newCandles)
		if err != nil {
			log.Printf("failed to upsert candles for %s: %v", sym, err)
		}
		totalInserted += int64(n)
	}

	return totalInserted, nil
}

func (s *Service) failRun(run *model.IngestRun, msg string) error {
	run.FinishedAt = time.Now().Unix()
	run.Status = "failed"
	run.ErrorMessage = msg
	db.UpdateIngestRun(run)
	return fmt.Errorf(msg)
}
