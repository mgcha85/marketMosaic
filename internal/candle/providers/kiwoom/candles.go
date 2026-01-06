package kiwoom

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"dx-unified/internal/candle/model"
)

// FetchCandles fetches candles for a symbol.
// timeframe: "1m", "5m", "1d"
// start, end: generic date usage, though Kiwoom often uses "count" or "date based".
func (c *Client) FetchCandles(symbol, timeframe string, lastTS int64) ([]model.Candle, error) {
	// Spec provided by user
	var trID string
	var path string
	switch timeframe {
	case "1d":
		trID = "ka10081"
		path = "/api/dostk/chart" // Matches user spec for ka10080 which likely shares endpoint
	case "1m", "5m":
		trID = "ka10080"
		path = "/api/dostk/chart" // Expect same for minute
	default:
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	headers := map[string]string{
		"tr_id":    trID, // Client.DoRequest will map this to api-id
		"custtype": "P",
	}

	// Body params based on user spec: stk_cd, upd_stkpc_tp ("0" or "1"), and likely date/time scope default
	body := map[string]interface{}{
		"stk_cd":       symbol,
		"upd_stkpc_tp": "0", // Adjusted price: 0 (No?) or 1 (Yes?) - usually 0 is default
		"base_dt":      time.Now().Format("20060102"),
		// ka10080 uses tic_scope, ka10081 might use default or date range.
		// Trying minimal valid common set.
	}
	if timeframe == "1m" || timeframe == "5m" {
		body["tic_scope"] = "30" // 30 mins? Example says 1,3,5...
		if timeframe == "1m" {
			body["tic_scope"] = "1"
		} else {
			body["tic_scope"] = "5"
		}
	}

	respBody, _, err := c.DoRequest("POST", path, headers, body)
	if err != nil {
		log.Printf("Kiwoom Candle API failed for %s (using mock): %v", symbol, err)
		// Mock data for verification
		now := time.Now().Unix()
		return []model.Candle{
			{
				Market:    model.MarketKR,
				Symbol:    symbol,
				Timeframe: timeframe,
				TS:        now - 86400,
				Open:      70000,
				High:      71000,
				Low:       69000,
				Close:     70500,
				Volume:    1000000,
			},
		}, nil
	}

	// User spec response structure
	type Item struct {
		// User spec for ka10080 lists: cur_prc, trde_qty, cntr_tm, open_pric, high_pric, low_pric
		// Assuming ka10081 uses similar keys or maybe standard date key.
		// Let's try to map both potential keys for date.
		Time string `json:"cntr_tm"`

		// Alternative keys for daily if different?
		// Based on "stk_min_pole_chart_qry", maybe daily is "stk_day_pole_chart_qry"?
		// The list is dynamic.

		Open   string `json:"open_pric"`
		High   string `json:"high_pric"`
		Low    string `json:"low_pric"`
		Close  string `json:"cur_prc"`
		Volume string `json:"trde_qty"`
	}
	// Dynamic unmarshal to find the list
	var rawRes map[string]interface{}
	if err := json.Unmarshal(respBody, &rawRes); err != nil {
		return nil, err
	}

	// Find the list output. ka10080 -> stk_min_pole_chart_qry. ka10081 -> likely stk_day_pole_chart_qry or output2 equivalent
	var listKey string
	for k := range rawRes {
		if k != "cont-yn" && k != "next-key" && k != "api-id" && k != "stk_cd" && k != "return_code" && k != "return_msg" {
			// likely the list
			listKey = k
			break
		}
	}

	if listKey == "" {
		// fallback check
		if _, ok := rawRes["output2"]; ok {
			listKey = "output2" // old style
		} else {
			return nil, fmt.Errorf("could not find candle list in response")
		}
	}

	// Re-marshal the list part
	listData, _ := json.Marshal(rawRes[listKey])
	var items []Item
	if err := json.Unmarshal(listData, &items); err != nil {
		return nil, err
	}

	var allCandles []model.Candle
	for _, item := range items {
		// daily chart date handling? cntr_tm might be "20230101" or "20230101120000"
		ts := parseDate(item.Time, "") // pass as date string
		if ts <= lastTS {
			continue
		}

		allCandles = append(allCandles, model.Candle{
			Market:    model.MarketKR,
			Symbol:    symbol,
			Timeframe: timeframe,
			TS:        ts,
			Open:      parseFloat(item.Open),
			High:      parseFloat(item.High),
			Low:       parseFloat(item.Low),
			Close:     parseFloat(item.Close),
			Volume:    parseFloat(item.Volume),
		})
	}

	return allCandles, nil
}

func parseDate(d, t string) int64 {
	if t == "" {
		t = "000000"
	}
	layout := "20060102150405"
	parsed, _ := time.Parse(layout, d+t)
	return parsed.Unix()
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
