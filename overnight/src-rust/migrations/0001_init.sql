CREATE TABLE IF NOT EXISTS signals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    signal_date TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    theme_idx INTEGER,
    theme_name TEXT,
    score REAL NOT NULL,
    score_theme REAL,
    score_news REAL,
    score_technical REAL,
    score_risk REAL,
    reasoning TEXT NOT NULL,
    entry_price REAL,
    change_rate REAL,
    ma5 REAL,
    ma5_support INTEGER DEFAULT 0,
    news_count INTEGER DEFAULT 0,
    news_titles TEXT DEFAULT '[]',
    explanation_json TEXT,
    structured_reasoning TEXT DEFAULT '{}',
    status TEXT DEFAULT 'pending',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(signal_date, code)
);

CREATE TABLE IF NOT EXISTS backtest_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_at TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT NOT NULL,
    min_score REAL DEFAULT 8.0,
    total_trades INTEGER DEFAULT 0,
    win_count INTEGER DEFAULT 0,
    loss_count INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    avg_return REAL DEFAULT 0,
    total_return REAL DEFAULT 0,
    max_drawdown REAL DEFAULT 0,
    status TEXT DEFAULT 'running',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS backtest_trades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    theme_name TEXT,
    entry_date TEXT NOT NULL,
    exit_date TEXT NOT NULL,
    entry_price REAL NOT NULL,
    exit_price REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    score REAL,
    score_theme REAL,
    score_news REAL,
    score_technical REAL,
    news_count INTEGER,
    ma5_support INTEGER,
    explanation_summary TEXT,
    explanation_json TEXT,
    FOREIGN KEY (run_id) REFERENCES backtest_runs(id)
);

CREATE TABLE IF NOT EXISTS backtest_candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    entry_date TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    theme_name TEXT,
    status TEXT NOT NULL,
    passed_score INTEGER DEFAULT 0,
    was_traded INTEGER DEFAULT 0,
    score REAL NOT NULL,
    score_theme REAL DEFAULT 0,
    score_news REAL DEFAULT 0,
    score_technical REAL DEFAULT 0,
    score_risk REAL DEFAULT 0,
    change_rate REAL,
    news_count INTEGER DEFAULT 0,
    ma5_support INTEGER DEFAULT 0,
    reasoning TEXT NOT NULL,
    structured_reasoning_json TEXT DEFAULT '{}',
    explanation_json TEXT DEFAULT '{}',
    strategy_checkpoints_json TEXT DEFAULT '[]',
    rejection_reasons_json TEXT DEFAULT '[]',
    exit_date TEXT,
    exit_price REAL,
    pnl_pct REAL,
    theme_rank INTEGER,
    theme_stock_count INTEGER,
    theme_positive_count INTEGER,
    history_json TEXT DEFAULT '[]',
    related_news_json TEXT DEFAULT '[]',
    candle_signal_json TEXT,
    FOREIGN KEY (run_id) REFERENCES backtest_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_signals_date ON signals(signal_date);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_created_at ON backtest_runs(created_at);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_run_id ON backtest_trades(run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_candidates_run_date ON backtest_candidates(run_id, entry_date);
