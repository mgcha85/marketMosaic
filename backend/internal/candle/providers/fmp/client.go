package fmp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"dx-unified/internal/candle/model"
)

const DefaultBaseURL = "https://financialmodelingprep.com/api/v3"

type Client struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: DefaultBaseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type ScreenerItem struct {
	Symbol      string  `json:"symbol"`
	CompanyName string  `json:"companyName"`
	MarketCap   float64 `json:"marketCap"`
	Sector      string  `json:"sector"`
	Industry    string  `json:"industry"`
	Beta        float64 `json:"beta"`
	Price       float64 `json:"price"`
	Exchange    string  `json:"exchangeShortName"` // e.g., NASDAQ, NYSE
	Country     string  `json:"country"`
	IsEtf       bool    `json:"isEtf"`
}

// FetchUniverse fetches the list of US stocks with market cap > minMcap.
// Uses /stock-screener endpoint.
func (c *Client) FetchUniverse(minMcap float64) ([]model.Instrument, error) {
	// Endpoint: /stock-screener?marketCapMoreThan=...&exchange=NASDAQ,NYSE&apikey=...
	// Also restrict country to US

	// Exchanges: "NASDAQ,NYSE,AMEX" or standard US exchanges.
	exchanges := "NASDAQ,NYSE,AMEX"

	u, err := url.Parse(c.BaseURL + "/stock-screener")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("apikey", c.APIKey)
	q.Set("marketCapMoreThan", fmt.Sprintf("%.0f", minMcap))
	q.Set("exchange", exchanges)
	q.Set("country", "US")
	q.Set("limit", "10000") // FMP limit? Paging?
	// FMP screener limit default is often low. We might need to handle paging if supported or just max it.
	// Documentation says limit param exists.

	u.RawQuery = q.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("FMP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP API error: %s", resp.Status)
	}

	var items []ScreenerItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode FMP response: %w", err)
	}

	var instruments []model.Instrument
	now := time.Now().Unix()
	for _, item := range items {
		instruments = append(instruments, model.Instrument{
			Market:      model.MarketUS,
			Symbol:      item.Symbol,
			Name:        item.CompanyName,
			Exchange:    item.Exchange,
			Currency:    "USD",
			MarketCap:   item.MarketCap,
			MarketCapTS: now, // Approximate, snapshot time
			IsActive:    true,
			UpdatedAt:   now,
		})
	}

	return instruments, nil
}
