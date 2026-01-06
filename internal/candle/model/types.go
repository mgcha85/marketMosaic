package model

// Market constants
const (
	MarketKR = "KR"
	MarketUS = "US"
)

// Instrument represents a stock or symbol.
type Instrument struct {
	Market      string  `json:"market"` // KR | US
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Exchange    string  `json:"exchange"`
	Currency    string  `json:"currency"`
	MarketCap   float64 `json:"market_cap"`
	MarketCapTS int64   `json:"market_cap_ts"` // UTC epoch sec
	IsActive    bool    `json:"is_active"`
	UpdatedAt   int64   `json:"updated_at"` // UTC epoch sec
}

// UniverseSnapshot represents a daily snapshot of the universe.
type UniverseSnapshot struct {
	YMD          string `json:"ymd"`
	Market       string `json:"market"`
	MarketCapMin float64 `json:"market_cap_min"`
	SymbolsJSON  string `json:"symbols_json"`
	CreatedAt    int64  `json:"created_at"`
}

// Candle represents a single OHLCV bar.
type Candle struct {
	Market     string  `json:"market"`
	Symbol     string  `json:"symbol"`
	Timeframe  string  `json:"timeframe"` // 1m, 5m, 1d
	TS         int64   `json:"ts"`        // Open time, UTC epoch sec
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Close      float64 `json:"close"`
	Volume     float64 `json:"volume"`
	VWAP       float64 `json:"vwap"`
	TradeCount int64   `json:"trade_count"`
}

// IngestRun represents an execution log.
type IngestRun struct {
	ID           int64  `json:"id"`
	StartedAt    int64  `json:"started_at"`
	FinishedAt   int64  `json:"finished_at"`
	Market       string `json:"market"`
	Job          string `json:"job"` // universe | candles
	Timeframe    string `json:"timeframe"`
	SymbolsCount int64  `json:"symbols_count"`
	InsertedRows int64  `json:"inserted_rows"`
	Status       string `json:"status"` // success | failed | partial
	ErrorMessage string `json:"error_message"`
}
