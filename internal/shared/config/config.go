package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the unified application
type Config struct {
	// Server
	Port string

	// SQLite Paths
	DartDBPath  string
	JudalDBPath string

	// Candle Data (Parquet/Hive Partition)
	CandleDataDir string

	// Meilisearch (News)
	MeiliHost   string
	MeiliAPIKey string

	// DART API
	DartAPIKey string

	// Storage
	StorageDir string

	// Kiwoom (KR Stock)
	KiwoomAppKey    string
	KiwoomAppSecret string
	KiwoomBaseURL   string

	// Alpaca (US Stock)
	AlpacaAPIKey    string
	AlpacaAPISecret string

	// FMP (US Universe)
	FMPAPIKey string

	// Naver News
	NaverClientID     string
	NaverClientSecret string

	// NewsAPI
	NewsAPIKey string

	// Judal
	CrawlDelay int
}

// Load reads configuration from environment variables
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DartDBPath:        getEnv("DART_DB_PATH", "./data/dart.db"),
		JudalDBPath:       getEnv("JUDAL_DB_PATH", "./data/judal.db"),
		CandleDataDir:     getEnv("CANDLE_DATA_DIR", "./data/candles"),
		MeiliHost:         getEnv("MEILI_HOST", "http://localhost:7700"),
		MeiliAPIKey:       getEnv("MEILI_API_KEY", "masterKey"),
		DartAPIKey:        os.Getenv("DART_API_KEY"),
		StorageDir:        getEnv("STORAGE_DIR", "./storage"),
		KiwoomAppKey:      os.Getenv("KIWOOM_APP_KEY"),
		KiwoomAppSecret:   os.Getenv("KIWOOM_APP_SECRET"),
		KiwoomBaseURL:     getEnv("KIWOOM_BASE_URL", "https://api.kiwoom.com"),
		AlpacaAPIKey:      os.Getenv("ALPACA_API_KEY"),
		AlpacaAPISecret:   os.Getenv("ALPACA_API_SECRET"),
		FMPAPIKey:         os.Getenv("FMP_API_KEY"),
		NaverClientID:     os.Getenv("NAVER_CLIENT_ID"),
		NaverClientSecret: os.Getenv("NAVER_CLIENT_SECRET"),
		NewsAPIKey:        os.Getenv("NEWSAPI_KEY"),
		CrawlDelay:        1500,
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
