package alpaca

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dx-unified/internal/candle/model"
)

const (
	DefaultBaseURL = "https://data.alpaca.markets"
	MaxSymbols     = 100 // Alpaca often limits symbols per request in free tier or generic URL length limits
)

type Client struct {
	APIKey    string
	APISecret string
	BaseURL   string
	client    *http.Client
}

func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		APIKey:    apiKey,
		APISecret: apiSecret,
		BaseURL:   DefaultBaseURL,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

type Bar struct {
	T  time.Time `json:"t"` // Timestamp
	O  float64   `json:"o"`
	H  float64   `json:"h"`
	L  float64   `json:"l"`
	C  float64   `json:"c"`
	V  float64   `json:"v"`
	N  int64     `json:"n"` // Trade count
	VW float64   `json:"vw"`
}

type BarsResponse struct {
	Bars          map[string][]Bar `json:"bars"`
	NextPageToken string           `json:"next_page_token"`
}

// FetchMultiBars fetches bars for multiple symbols.
// timeframe: "1Min", "5Min", "1Day" (Alpaca format)
// start, end: RFC3339 strings
func (c *Client) FetchMultiBars(symbols []string, timeframe string, start, end string) (map[string][]model.Candle, error) {
	// Chunk symbols
	results := make(map[string][]model.Candle)

	// Helper to normalize timeframe to Alpaca format if needed
	// model uses "1m", "5m", "1d"
	alpacaTF := timeframe
	switch timeframe {
	case "1m":
		alpacaTF = "1Min"
	case "5m":
		alpacaTF = "5Min"
	case "1d":
		alpacaTF = "1Day"
	}

	for i := 0; i < len(symbols); i += MaxSymbols {
		j := i + MaxSymbols
		if j > len(symbols) {
			j = len(symbols)
		}
		chunk := symbols[i:j]

		// Fetch chunk
		chunkRes, err := c.fetchBarsRaw(chunk, alpacaTF, start, end)
		if err != nil {
			return nil, err
		}

		// Convert and merge
		for sym, bars := range chunkRes {
			var candles []model.Candle
			for _, b := range bars {
				candles = append(candles, model.Candle{
					Market:     model.MarketUS,
					Symbol:     sym,
					Timeframe:  timeframe, // Use internal format "1m"
					TS:         b.T.Unix(),
					Open:       b.O,
					High:       b.H,
					Low:        b.L,
					Close:      b.C,
					Volume:     b.V,
					VWAP:       b.VW,
					TradeCount: b.N,
				})
			}
			results[sym] = candles
		}
	}

	return results, nil
}

func (c *Client) fetchBarsRaw(symbols []string, timeframe, start, end string) (map[string][]Bar, error) {
	symbolStr := strings.Join(symbols, ",")
	allBars := make(map[string][]Bar)
	pageToken := ""

	for {
		u, _ := url.Parse(c.BaseURL + "/v2/stocks/bars")
		q := u.Query()
		q.Set("symbols", symbolStr)
		q.Set("timeframe", timeframe)
		if start != "" {
			q.Set("start", start)
		}
		if end != "" {
			q.Set("end", end)
		}
		q.Set("limit", "10000")    // Max limit
		q.Set("adjustment", "raw") // or split? User usually wants adjusted, but "raw" is safer for pure candle match?
		// Alpaca default is "raw" for free logic or "split"?
		// Usually adjustment='split' is better for analysis.
		q.Set("adjustment", "split")

		if pageToken != "" {
			q.Set("page_token", pageToken)
		}

		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("APCA-API-KEY-ID", c.APIKey)
		req.Header.Set("APCA-API-SECRET-KEY", c.APISecret)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("alpaca request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("alpaca API error: %s", resp.Status)
		}

		var res BarsResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("failed to decode alpaca response: %w", err)
		}

		for sym, bars := range res.Bars {
			allBars[sym] = append(allBars[sym], bars...)
		}

		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}

	return allBars, nil
}
