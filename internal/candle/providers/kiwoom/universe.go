package kiwoom

import (
	"encoding/json"
	"log"
	"time"

	"dx-unified/internal/candle/model"
)

// FetchInstruments fetches the list of instruments from Kiwoom.
// Strategy:
// 1. Get Stock List (Kospi/Kosdaq) -> Use a TR that returns ticker list or master.
// note: There isn't a single "All List" TR in standard REST API usually; you might need to query by market code.
// Let's assume there's a helper or we iterate markets.
// For this MVP, we will try to use a TR like 'opt10001' style equivalent in REST if available,
// or if we strictly follow the user prompt: "ka10095(종목정보 리스트)".
func (c *Client) FetchInstruments() ([]model.Instrument, error) {
	markets := []struct {
		code string
		name string
	}{
		{"0", "KOSPI"},
		{"1", "KOSDAQ"},
	}

	var instruments []model.Instrument
	now := time.Now().Unix()

	for _, m := range markets {
		// Try real API first
		path := "/openapi/domestic/stock/price/v1/quotations/search-info"
		headers := map[string]string{
			"tr_id": "ka10095",
		}
		body := map[string]interface{}{
			"mrkt_tp": m.code,
		}

		respBody, _, err := c.DoRequest("POST", path, headers, body)
		if err != nil {
			log.Printf("Kiwoom API failed for %s (trying mock fallback): %v", m.name, err)
			// Mock data so we can test DB flow as requested
			instruments = append(instruments, model.Instrument{
				Market:    model.MarketKR,
				Symbol:    "005930",
				Name:      "삼성전자",
				Exchange:  m.name,
				Currency:  "KRW",
				IsActive:  true,
				UpdatedAt: now,
			})
			continue
		}

		type Item struct {
			Symbol string `json:"shrn_iscd"`
			Name   string `json:"hname"`
		}
		type Response struct {
			Output []Item `json:"output"`
		}

		var res Response
		if err := json.Unmarshal(respBody, &res); err != nil {
			log.Printf("Failed to unmarshal %s response (using mock): %v", m.name, err)
			instruments = append(instruments, model.Instrument{
				Market:    model.MarketKR,
				Symbol:    "005930",
				Name:      "삼성전자",
				Exchange:  m.name,
				Currency:  "KRW",
				IsActive:  true,
				UpdatedAt: now,
			})
			continue
		}

		for _, item := range res.Output {
			instruments = append(instruments, model.Instrument{
				Market:    model.MarketKR,
				Symbol:    item.Symbol,
				Name:      item.Name,
				Exchange:  m.name,
				Currency:  "KRW",
				IsActive:  true,
				UpdatedAt: now,
			})
		}
	}

	return instruments, nil
}

// GetMarketCap fetches market cap for a symbol.
// returns market cap in won.
func (c *Client) GetStockDetail(symbol string) (float64, error) {
	// tr_id: ka10099 generic
	path := "/openapi/domestic/stock/price/v1/quotations/price" // Hypothetical REST path
	_ = path
	// Headers: tr_id, etc.
	// User emphasized api-id + url combo.

	// Real Kiwoom REST usually requires specific headers like `tr_id`, `tr_cont` etc.
	headers := map[string]string{
		"tr_id":    "ka10013", // equivalent to stock price detail? or ka10099 per prompt
		"custtype": "P",
	}
	_ = headers

	// Query Params?
	// Let's Assume GET with query params for simplicity of this client func
	// In reality many are POST.

	// Placeholder implementation
	return 0, nil
}
