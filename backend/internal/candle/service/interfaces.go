package service

import "dx-unified/internal/candle/model"

// UniverseProvider defines the interface for fetching the universe.
type UniverseProvider interface {
	FetchUniverse(minMcap float64) ([]model.Instrument, error)
	FetchInstruments() ([]model.Instrument, error) // for Kiwoom style
}

// CandleProvider defines the interface for fetching candles.
type CandleProvider interface {
	FetchCandles(symbol, timeframe string, lastTS int64) ([]model.Candle, error)
}

// MultiCandleProvider for Alpaca
type MultiCandleProvider interface {
	FetchMultiBars(symbols []string, timeframe, start, end string) (map[string][]model.Candle, error)
}
