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

// MinuteCandle represents a minute OHLCV candle
type MinuteCandle struct {
	Time   string  `json:"datetime"` // Matches API: "2026-01-07 09:05:00.000000000"
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"` // API returns float like 4502414.0
}

// MinuteCandleResponse is the response from /stocks/minute-ohlcv
type MinuteCandleResponse struct {
	Count int            `json:"count"`
	Data  []MinuteCandle `json:"data"`
}

// StockInfo represents a stock in the list
type StockInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// StockListResponse is the response from /stocks/list
type StockListResponse []StockInfo

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

// GetMinuteCandles fetches minute OHLCV data for a stock
func (c *Client) GetMinuteCandles(stockCode, startDateTime, endDateTime string) (*MinuteCandleResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("kiwoom REST API not configured")
	}

	url := fmt.Sprintf("%s/stocks/minute-ohlcv?code=%s", c.baseURL, stockCode)

	if startDateTime != "" {
		url += fmt.Sprintf("&start_datetime=%s", startDateTime)
	}
	if endDateTime != "" {
		url += fmt.Sprintf("&end_datetime=%s", endDateTime)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch minute candles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minute candle API returned status %d", resp.StatusCode)
	}

	var result MinuteCandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode minute candle response: %w", err)
	}

	return &result, nil
}

// GetStockList fetches the list of all stocks
func (c *Client) GetStockList() (StockListResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("kiwoom REST API not configured")
	}

	url := fmt.Sprintf("%s/stocks/list", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stock list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stock list API returned status %d", resp.StatusCode)
	}

	var result StockListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode stock list response: %w", err)
	}

	return result, nil
}

// =====================
// New Methods for Minute API
// =====================

// MinuteAvailableResponse response from /api/stocks/minute-available
type MinuteAvailableResponse struct {
	Count int      `json:"count"`
	Data  []string `json:"data"`
}

// GetAvailableMinuteStocks fetches the list of stocks available for minute resolution
func (c *Client) GetAvailableMinuteStocks() (*MinuteAvailableResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("kiwoom REST API not configured")
	}

	url := fmt.Sprintf("%s/api/stocks/minute-available", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch minute available list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minute available API returned status %d", resp.StatusCode)
	}

	var result MinuteAvailableResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode minute available response: %w", err)
	}

	return &result, nil
}
