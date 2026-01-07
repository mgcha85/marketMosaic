package config

import (
	"os"
	"strconv"
)

// Config 애플리케이션 설정
type Config struct {
	ServerAddr   string
	DatabasePath string
	CrawlDelay   int // milliseconds
	AutoCrawl    bool
}

// Load 환경변수에서 설정 로드
func Load() *Config {
	config := &Config{
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/judal.db"),
		CrawlDelay:   getEnvInt("CRAWL_DELAY", 1500),
		AutoCrawl:    getEnvBool("AUTO_CRAWL", false),
	}
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return defaultValue
}
