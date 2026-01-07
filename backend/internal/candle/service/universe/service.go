package universe

import (
	"encoding/json"
	"fmt"
	"time"

	db "dx-unified/internal/candle/database"
	"dx-unified/internal/candle/model"
)

type ValidationProvider interface {
	// FMP
	FetchUniverse(minMcap float64) ([]model.Instrument, error)
}

type ListProvider interface {
	// Kiwoom
	FetchInstruments() ([]model.Instrument, error)
}

// Service handles universe generation.
type Service struct {
	usProvider ValidationProvider
	krProvider ListProvider
}

// NewService accepts flexible providers.
// We can use a pattern where we pass nil for one.
func NewService(us ValidationProvider, kr ListProvider) *Service {
	return &Service{
		usProvider: us,
		krProvider: kr,
	}
}

// BuildAndSave builds the universe for the given market and saves it.
func (s *Service) BuildAndSave(market string, minMcap float64) error {
	var instruments []model.Instrument
	var err error

	if market == model.MarketUS {
		if s.usProvider == nil {
			return fmt.Errorf("US provider not configured")
		}
		instruments, err = s.usProvider.FetchUniverse(minMcap)
	} else if market == model.MarketKR {
		if s.krProvider == nil {
			return fmt.Errorf("KR provider not configured")
		}
		instruments, err = s.krProvider.FetchInstruments()

		// Filter by Mcap
		if err == nil {
			var filtered []model.Instrument
			for _, inst := range instruments {
				if inst.MarketCap >= minMcap {
					filtered = append(filtered, inst)
				}
			}
			instruments = filtered
		}
	} else {
		return fmt.Errorf("unsupported market: %s", market)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch universe: %w", err)
	}

	// 1. Upsert Instruments
	if err := db.UpsertInstruments(instruments); err != nil {
		return fmt.Errorf("failed to upsert instruments: %w", err)
	}

	// 2. Save Snapshot
	// Extract symbols for JSON
	var symbols []string
	for _, inst := range instruments {
		symbols = append(symbols, inst.Symbol)
	}
	symbolsJSON, _ := json.Marshal(symbols)

	ymd := time.Now().Format("2006-01-02")
	snapshot := model.UniverseSnapshot{
		YMD:          ymd,
		Market:       market,
		MarketCapMin: minMcap,
		SymbolsJSON:  string(symbolsJSON),
		CreatedAt:    time.Now().Unix(),
	}

	if err := db.SaveUniverseSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	fmt.Printf("Successfully built universe for %s. Count: %d\n", market, len(instruments))

	return nil
}
