package database

import "log"

const schemaSQL = `
-- 테마 테이블
CREATE TABLE IF NOT EXISTS themes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theme_idx INTEGER UNIQUE NOT NULL,
    name TEXT NOT NULL,
    stock_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 종목 테이블 (최신 데이터)
CREATE TABLE IF NOT EXISTS stocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    market TEXT,
    current_price INTEGER,
    price_change INTEGER,
    change_rate REAL,
    three_day_sum REAL,
    high_52w INTEGER,
    low_52w INTEGER,
    change_rate_52w_up REAL,
    change_rate_52w_down REAL,
    neglect_index_52w REAL,
    high_3y INTEGER,
    low_3y INTEGER,
    change_rate_3y_up REAL,
    change_rate_3y_down REAL,
    neglect_index_3y REAL,
    price_index_3y REAL,
    expected_return REAL,
    pbr REAL,
    per REAL,
    eps INTEGER,
    market_cap INTEGER,
    volume_index REAL,
    volume_index_7d REAL,
    buffett_choice INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 종목 히스토리 테이블 (일별 스냅샷)
CREATE TABLE IF NOT EXISTS stock_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    crawl_date DATE NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    market TEXT,
    current_price INTEGER,
    price_change INTEGER,
    change_rate REAL,
    three_day_sum REAL,
    high_52w INTEGER,
    low_52w INTEGER,
    neglect_index_52w REAL,
    price_index_3y REAL,
    expected_return REAL,
    pbr REAL,
    per REAL,
    eps INTEGER,
    market_cap INTEGER,
    volume_index REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(crawl_date, code)
);

-- 테마-종목 매핑 테이블
CREATE TABLE IF NOT EXISTS theme_stocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theme_idx INTEGER NOT NULL,
    stock_code TEXT NOT NULL,
    UNIQUE(theme_idx, stock_code)
);

-- 크롤링 로그 테이블
CREATE TABLE IF NOT EXISTS crawl_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    crawl_date DATE NOT NULL,
    crawl_type TEXT NOT NULL,
    themes_count INTEGER DEFAULT 0,
    stocks_count INTEGER DEFAULT 0,
    duration_seconds REAL,
    status TEXT DEFAULT 'completed',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 인덱스
CREATE INDEX IF NOT EXISTS idx_stocks_code ON stocks(code);
CREATE INDEX IF NOT EXISTS idx_stocks_market ON stocks(market);
CREATE INDEX IF NOT EXISTS idx_stocks_change_rate ON stocks(change_rate);
CREATE INDEX IF NOT EXISTS idx_stocks_market_cap ON stocks(market_cap);
CREATE INDEX IF NOT EXISTS idx_theme_stocks_theme ON theme_stocks(theme_idx);
CREATE INDEX IF NOT EXISTS idx_theme_stocks_stock ON theme_stocks(stock_code);
CREATE INDEX IF NOT EXISTS idx_stock_history_date ON stock_history(crawl_date);
CREATE INDEX IF NOT EXISTS idx_stock_history_code ON stock_history(code);
CREATE INDEX IF NOT EXISTS idx_crawl_logs_date ON crawl_logs(crawl_date);
`

func initSchema() error {
	_, err := DB.Exec(schemaSQL)
	if err != nil {
		log.Printf("Error initializing schema: %v", err)
		return err
	}
	log.Println("Schema initialized successfully")
	return nil
}
