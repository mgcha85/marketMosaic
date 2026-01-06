package models

import "time"

// Theme 테마 정보
type Theme struct {
	ID         int64     `json:"id"`
	ThemeIdx   int       `json:"theme_idx"`
	Name       string    `json:"name"`
	StockCount int       `json:"stock_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ThemeWithStocks 종목 목록이 포함된 테마
type ThemeWithStocks struct {
	Theme
	Stocks []Stock `json:"stocks,omitempty"`
}
