package main

import (
	"database/sql"
	"log"
	"time"

	dartDB "dx-unified/internal/dart/database"
	dartModels "dx-unified/internal/dart/models"
	judalDB "dx-unified/internal/judal/database"
	judalModels "dx-unified/internal/judal/models"
)

func main() {
	log.Println("Starting seeding...")

	// 1. DART Seeding
	if err := dartDB.InitDB("data/dart.db"); err != nil {
		log.Fatalf("Failed to init DART DB: %v", err)
	}
	dbDart := dartDB.GetDB()

	// Samsung Electronics
	corp := dartModels.Corp{
		CorpCode:   "00126380",
		CorpName:   "삼성전자",
		StockCode:  "005930",
		ModifiedAt: time.Now(),
	}
	dbDart.FirstOrCreate(&corp, dartModels.Corp{CorpCode: "00126380"})

	// Sample Filings
	filings := []dartModels.Filing{
		{
			RceptNo:   "20240101000001",
			CorpCode:  "00126380",
			CorpName:  "삼성전자",
			ReportNm:  "현금배당결정",
			RceptDt:   "20240101",
			FlrNm:     "삼성전자",
			Rm:        "유",
			CreatedAt: time.Now(),
		},
		{
			RceptNo:   "20240105000002",
			CorpCode:  "00126380",
			CorpName:  "삼성전자",
			ReportNm:  "영업잠정실적발표",
			RceptDt:   "20240105",
			FlrNm:     "삼성전자",
			Rm:        "코",
			CreatedAt: time.Now(),
		},
	}
	for _, f := range filings {
		dbDart.FirstOrCreate(&f, dartModels.Filing{RceptNo: f.RceptNo})
	}
	log.Println("DART mock data seeded.")

	// 2. Judal Seeding
	if err := judalDB.InitDB("data/judal.db"); err != nil {
		log.Fatalf("Failed to init Judal DB: %v", err)
	}
	repo := judalDB.NewRepository()

	// Theme
	theme := &judalModels.Theme{
		ThemeIdx:   100,
		Name:       "반도체 대표주",
		StockCount: 2,
	}
	if err := repo.UpsertTheme(theme); err != nil {
		log.Printf("Failed to upsert theme: %v", err)
	}

	// Stocks
	samsung := &judalModels.Stock{
		Code:         "005930",
		Name:         "삼성전자",
		Market:       "KOSPI",
		CurrentPrice: sql.NullInt64{Int64: 75000, Valid: true},
		ChangeRate:   sql.NullFloat64{Float64: 1.5, Valid: true},
		PER:          sql.NullFloat64{Float64: 12.5, Valid: true},
		PBR:          sql.NullFloat64{Float64: 1.3, Valid: true},
		MarketCap:    sql.NullInt64{Int64: 450000000000000, Valid: true},
	}
	if err := repo.UpsertStock(samsung); err != nil {
		log.Printf("Failed to upsert stock Samsung: %v", err)
	}

	skhynix := &judalModels.Stock{
		Code:         "000660",
		Name:         "SK하이닉스",
		Market:       "KOSPI",
		CurrentPrice: sql.NullInt64{Int64: 140000, Valid: true},
		ChangeRate:   sql.NullFloat64{Float64: 2.1, Valid: true},
		PER:          sql.NullFloat64{Float64: 15.0, Valid: true},
		PBR:          sql.NullFloat64{Float64: 1.8, Valid: true},
	}
	if err := repo.UpsertStock(skhynix); err != nil {
		log.Printf("Failed to upsert stock SK: %v", err)
	}

	// Mapping
	if err := repo.AddThemeStock(100, "005930"); err != nil {
		log.Printf("Failed to map theme: %v", err)
	}
	if err := repo.AddThemeStock(100, "000660"); err != nil {
		log.Printf("Failed to map theme: %v", err)
	}

	log.Println("Judal mock data seeded.")
	log.Println("Done.")
}
