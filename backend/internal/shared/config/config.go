package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the unified application
type Config struct {
	// Server
	Port string `json:"port"`

	// SQLite Paths
	DartDBPath  string `json:"dart_db_path"`
	JudalDBPath string `json:"judal_db_path"`

	// Candle Data (Parquet/Hive Partition)
	CandleDataDir string `json:"candle_data_dir"`

	// Meilisearch (News)
	MeiliHost   string `json:"meili_host"`
	MeiliAPIKey string `json:"meili_api_key"`

	// DART API
	DartAPIKey string `json:"dart_api_key"`

	// Storage
	StorageDir string `json:"storage_dir"`

	// Kiwoom (KR Stock)
	KiwoomAppKey     string `json:"kiwoom_app_key"`
	KiwoomAppSecret  string `json:"kiwoom_app_secret"`
	KiwoomBaseURL    string `json:"kiwoom_base_url"`
	KiwoomRestAPIURL string `json:"kiwoom_rest_api_url"`

	// Alpaca (US Stock)
	AlpacaAPIKey    string `json:"alpaca_api_key"`
	AlpacaAPISecret string `json:"alpaca_api_secret"`

	// FMP (US Universe)
	FMPAPIKey string `json:"fmp_api_key"`

	// Naver News
	NaverClientID     string `json:"naver_client_id"`
	NaverClientSecret string `json:"naver_client_secret"`

	// NewsAPI
	NewsAPIKey string `json:"newsapi_key"`

	// Judal
	CrawlDelay int `json:"crawl_delay"`

	// News Filtering
	NaverQueries       []string `json:"naver_queries"`
	EconKeywordsAllow  []string `json:"econ_keywords_allow"`
	EconKeywordsBlock  []string `json:"econ_keywords_block"`
	TitleKeywordsBlock []string `json:"title_keywords_block"`
	GenericNewsBlock   []string `json:"generic_news_block"`

	// News Fetch Interval (Cron expression)
	NewsFetchCron string `json:"news_fetch_cron"`
}

// Load reads configuration from environment variables and optional config.json
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
		KiwoomRestAPIURL:  getEnv("KIWOOM_REST_API_URL", "http://131.186.33.55:8083"),
		AlpacaAPIKey:      os.Getenv("ALPACA_API_KEY"),
		AlpacaAPISecret:   os.Getenv("ALPACA_API_SECRET"),
		FMPAPIKey:         os.Getenv("FMP_API_KEY"),
		NaverClientID:     os.Getenv("NAVER_CLIENT_ID"),
		NaverClientSecret: os.Getenv("NAVER_CLIENT_SECRET"),
		NewsAPIKey:        os.Getenv("NEWSAPI_KEY"),
		NewsFetchCron:     getEnv("NEWS_FETCH_CRON", "*/15 * * * *"),
		CrawlDelay:        1500,
		NaverQueries:      []string{"주식", "증시", "경제", "코스피", "코스닥"},
		EconKeywordsAllow: []string{"금리", "투자", "실적", "상장", "매수", "매도"},
		EconKeywordsBlock: []string{"부고", "인사", "결혼", "모집"},
	}

	// Try loading from data/config.json to override
	// configPath should be flexible based on where 'data' dir is relative to execution
	// Usually ./data/config.json in container
	configPath := "./data/config.json"

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.Open(configPath)
		if err == nil {
			defer file.Close()
			var jsonCfg Config
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&jsonCfg); err == nil {
				log.Println("Loaded configuration overrides from config.json")
				overrideConfig(cfg, &jsonCfg)
			} else {
				log.Printf("Failed to parse config.json: %v", err)
			}
		}
	}

	return cfg
}

func overrideConfig(base, override *Config) {
	// Helper to override only non-empty string fields
	// Integer fields (CrawlDelay) are overridden if non-zero

	if override.DartAPIKey != "" {
		base.DartAPIKey = override.DartAPIKey
	}
	if override.KiwoomAppKey != "" {
		base.KiwoomAppKey = override.KiwoomAppKey
	}
	if override.KiwoomAppSecret != "" {
		base.KiwoomAppSecret = override.KiwoomAppSecret
	}
	if override.KiwoomBaseURL != "" {
		base.KiwoomBaseURL = override.KiwoomBaseURL
	}
	if override.KiwoomRestAPIURL != "" {
		base.KiwoomRestAPIURL = override.KiwoomRestAPIURL
	}
	if override.AlpacaAPIKey != "" {
		base.AlpacaAPIKey = override.AlpacaAPIKey
	}
	if override.AlpacaAPISecret != "" {
		base.AlpacaAPISecret = override.AlpacaAPISecret
	}
	if override.FMPAPIKey != "" {
		base.FMPAPIKey = override.FMPAPIKey
	}
	if override.NaverClientID != "" {
		base.NaverClientID = override.NaverClientID
	}
	if override.NaverClientSecret != "" {
		base.NaverClientSecret = override.NaverClientSecret
	}
	if override.NewsAPIKey != "" {
		base.NewsAPIKey = override.NewsAPIKey
	}
	if override.NewsFetchCron != "" {
		base.NewsFetchCron = override.NewsFetchCron
	}
	if override.CrawlDelay > 0 {
		base.CrawlDelay = override.CrawlDelay
	}
	// Add more overrides as needed
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
