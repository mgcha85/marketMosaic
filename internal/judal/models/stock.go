package models

import (
	"database/sql"
	"time"
)

// Stock 종목 정보
type Stock struct {
	ID                int64           `json:"id"`
	Code              string          `json:"code"`
	Name              string          `json:"name"`
	Market            string          `json:"market"` // KOSPI or KOSDAQ
	CurrentPrice      sql.NullInt64   `json:"current_price"`
	PriceChange       sql.NullInt64   `json:"price_change"`
	ChangeRate        sql.NullFloat64 `json:"change_rate"`
	ThreeDaySum       sql.NullFloat64 `json:"three_day_sum"`
	High52W           sql.NullInt64   `json:"high_52w"`
	Low52W            sql.NullInt64   `json:"low_52w"`
	ChangeRate52WUp   sql.NullFloat64 `json:"change_rate_52w_up"`
	ChangeRate52WDown sql.NullFloat64 `json:"change_rate_52w_down"`
	NeglectIndex52W   sql.NullFloat64 `json:"neglect_index_52w"`
	High3Y            sql.NullInt64   `json:"high_3y"`
	Low3Y             sql.NullInt64   `json:"low_3y"`
	ChangeRate3YUp    sql.NullFloat64 `json:"change_rate_3y_up"`
	ChangeRate3YDown  sql.NullFloat64 `json:"change_rate_3y_down"`
	NeglectIndex3Y    sql.NullFloat64 `json:"neglect_index_3y"`
	PriceIndex3Y      sql.NullFloat64 `json:"price_index_3y"`
	ExpectedReturn    sql.NullFloat64 `json:"expected_return"`
	PBR               sql.NullFloat64 `json:"pbr"`
	PER               sql.NullFloat64 `json:"per"`
	EPS               sql.NullInt64   `json:"eps"`
	MarketCap         sql.NullInt64   `json:"market_cap"`
	VolumeIndex       sql.NullFloat64 `json:"volume_index"`
	VolumeIndex7D     sql.NullFloat64 `json:"volume_index_7d"`
	BuffettChoice     sql.NullInt64   `json:"buffett_choice"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// StockJSON JSON 직렬화를 위한 종목 구조체
type StockJSON struct {
	ID                int64     `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Market            string    `json:"market"`
	CurrentPrice      *int64    `json:"current_price,omitempty"`
	PriceChange       *int64    `json:"price_change,omitempty"`
	ChangeRate        *float64  `json:"change_rate,omitempty"`
	ThreeDaySum       *float64  `json:"three_day_sum,omitempty"`
	High52W           *int64    `json:"high_52w,omitempty"`
	Low52W            *int64    `json:"low_52w,omitempty"`
	ChangeRate52WUp   *float64  `json:"change_rate_52w_up,omitempty"`
	ChangeRate52WDown *float64  `json:"change_rate_52w_down,omitempty"`
	NeglectIndex52W   *float64  `json:"neglect_index_52w,omitempty"`
	High3Y            *int64    `json:"high_3y,omitempty"`
	Low3Y             *int64    `json:"low_3y,omitempty"`
	ChangeRate3YUp    *float64  `json:"change_rate_3y_up,omitempty"`
	ChangeRate3YDown  *float64  `json:"change_rate_3y_down,omitempty"`
	NeglectIndex3Y    *float64  `json:"neglect_index_3y,omitempty"`
	PriceIndex3Y      *float64  `json:"price_index_3y,omitempty"`
	ExpectedReturn    *float64  `json:"expected_return,omitempty"`
	PBR               *float64  `json:"pbr,omitempty"`
	PER               *float64  `json:"per,omitempty"`
	EPS               *int64    `json:"eps,omitempty"`
	MarketCap         *int64    `json:"market_cap,omitempty"`
	VolumeIndex       *float64  `json:"volume_index,omitempty"`
	VolumeIndex7D     *float64  `json:"volume_index_7d,omitempty"`
	BuffettChoice     *int64    `json:"buffett_choice,omitempty"`
	RelatedThemes     []string  `json:"related_themes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToJSON Stock을 StockJSON으로 변환
func (s *Stock) ToJSON() StockJSON {
	sj := StockJSON{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		Market:    s.Market,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}

	if s.CurrentPrice.Valid {
		sj.CurrentPrice = &s.CurrentPrice.Int64
	}
	if s.PriceChange.Valid {
		sj.PriceChange = &s.PriceChange.Int64
	}
	if s.ChangeRate.Valid {
		sj.ChangeRate = &s.ChangeRate.Float64
	}
	if s.ThreeDaySum.Valid {
		sj.ThreeDaySum = &s.ThreeDaySum.Float64
	}
	if s.High52W.Valid {
		sj.High52W = &s.High52W.Int64
	}
	if s.Low52W.Valid {
		sj.Low52W = &s.Low52W.Int64
	}
	if s.ChangeRate52WUp.Valid {
		sj.ChangeRate52WUp = &s.ChangeRate52WUp.Float64
	}
	if s.ChangeRate52WDown.Valid {
		sj.ChangeRate52WDown = &s.ChangeRate52WDown.Float64
	}
	if s.NeglectIndex52W.Valid {
		sj.NeglectIndex52W = &s.NeglectIndex52W.Float64
	}
	if s.High3Y.Valid {
		sj.High3Y = &s.High3Y.Int64
	}
	if s.Low3Y.Valid {
		sj.Low3Y = &s.Low3Y.Int64
	}
	if s.ChangeRate3YUp.Valid {
		sj.ChangeRate3YUp = &s.ChangeRate3YUp.Float64
	}
	if s.ChangeRate3YDown.Valid {
		sj.ChangeRate3YDown = &s.ChangeRate3YDown.Float64
	}
	if s.NeglectIndex3Y.Valid {
		sj.NeglectIndex3Y = &s.NeglectIndex3Y.Float64
	}
	if s.PriceIndex3Y.Valid {
		sj.PriceIndex3Y = &s.PriceIndex3Y.Float64
	}
	if s.ExpectedReturn.Valid {
		sj.ExpectedReturn = &s.ExpectedReturn.Float64
	}
	if s.PBR.Valid {
		sj.PBR = &s.PBR.Float64
	}
	if s.PER.Valid {
		sj.PER = &s.PER.Float64
	}
	if s.EPS.Valid {
		sj.EPS = &s.EPS.Int64
	}
	if s.MarketCap.Valid {
		sj.MarketCap = &s.MarketCap.Int64
	}
	if s.VolumeIndex.Valid {
		sj.VolumeIndex = &s.VolumeIndex.Float64
	}
	if s.VolumeIndex7D.Valid {
		sj.VolumeIndex7D = &s.VolumeIndex7D.Float64
	}
	if s.BuffettChoice.Valid {
		sj.BuffettChoice = &s.BuffettChoice.Int64
	}

	return sj
}

// ThemeStock 테마-종목 매핑
type ThemeStock struct {
	ID        int64  `json:"id"`
	ThemeIdx  int    `json:"theme_idx"`
	StockCode string `json:"stock_code"`
}
