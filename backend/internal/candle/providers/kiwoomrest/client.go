package kiwoomrest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is a client for the Kiwoom REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Kiwoom REST API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if the client is properly configured
func (c *Client) IsConfigured() bool {
	return c.baseURL != ""
}

// =====================
// Models
// =====================

// Fundamental represents fundamental data for a stock
type Fundamental struct {
	Date string  `json:"date"`
	EPS  float64 `json:"EPS"`
	PER  float64 `json:"PER"`
	PBR  float64 `json:"PBR"`
	BPS  float64 `json:"BPS,omitempty"`
	DIV  float64 `json:"DIV,omitempty"`
	DPS  float64 `json:"DPS,omitempty"`
}

// FundamentalResponse is the response from /stocks/{code}/fundamental
type FundamentalResponse struct {
	Count int           `json:"count"`
	Data  []Fundamental `json:"data"`
}

// DailyCandle represents a daily OHLCV candle
type DailyCandle struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// CandleResponse is the response from /stocks/{code}/period-ohlcv
type CandleResponse struct {
	Count int           `json:"count"`
	Data  []DailyCandle `json:"data"`
}

// =====================
// API Methods
// =====================

// GetFundamental fetches fundamental data for a stock
func (c *Client) GetFundamental(stockCode string) (*FundamentalResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("kiwoom REST API not configured")
	}

	url := fmt.Sprintf("%s/stocks/%s/fundamental", c.baseURL, stockCode)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fundamental: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fundamental API returned status %d", resp.StatusCode)
	}

	var result FundamentalResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode fundamental response: %w", err)
	}

	return &result, nil
}

// GetDailyCandles fetches daily OHLCV data for a stock
func (c *Client) GetDailyCandles(stockCode string, startDate, endDate string) (*CandleResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("kiwoom REST API not configured")
	}

	url := fmt.Sprintf("%s/stocks/%s/period-ohlcv", c.baseURL, stockCode)

	if startDate != "" || endDate != "" {
		url += "?"
		if startDate != "" {
			url += fmt.Sprintf("start_date=%s", startDate)
		}
		if endDate != "" {
			if startDate != "" {
				url += "&"
			}
			url += fmt.Sprintf("end_date=%s", endDate)
		}
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("candle API returned status %d", resp.StatusCode)
	}

	var result CandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode candle response: %w", err)
	}

	return &result, nil
}
