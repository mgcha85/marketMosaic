use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Signal {
    pub id: i64,
    pub signal_date: String,
    pub code: String,
    pub name: String,
    pub theme_idx: Option<i64>,
    pub theme_name: Option<String>,
    pub score: f64,
    pub score_theme: Option<f64>,
    pub score_news: Option<f64>,
    pub score_technical: Option<f64>,
    pub score_risk: Option<f64>,
    pub reasoning: String,
    pub entry_price: Option<f64>,
    pub change_rate: Option<f64>,
    pub ma5: Option<f64>,
    pub ma5_support: bool,
    pub news_count: i64,
    pub news_titles: Vec<String>,
    pub structured_reasoning: StructuredReasoning,
    pub status: String,
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GenerateSignalsResponse {
    pub date: String,
    pub generated_count: usize,
    pub signals: Vec<Signal>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SignalDetail {
    pub signal: Signal,
    pub history: Vec<StockHistoryPoint>,
    pub related_news: Vec<NewsArticle>,
    pub explanation: ScoreExplanation,
    pub candle_signal: Option<CandleSignal>,
    pub strategy_checkpoints: Vec<StrategyCheckpoint>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StockHistoryPoint {
    pub crawl_date: String,
    pub current_price: Option<i64>,
    pub change_rate: Option<f64>,
    pub market_cap: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NewsArticle {
    pub title: String,
    pub published_at: String,
    pub publisher: Option<String>,
    pub url: Option<String>,
    pub age_hours: Option<i64>,
}

#[derive(Debug, Clone)]
pub struct SignalCandidate {
    pub code: String,
    pub name: String,
    pub theme_idx: i64,
    pub theme_name: String,
    pub current_price: Option<i64>,
    pub change_rate: Option<f64>,
    pub market_cap: Option<i64>,
    pub volume_index: Option<f64>,
    pub theme_stock_count: Option<i64>,
    pub theme_rank: Option<i64>,
    pub theme_positive_count: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct CandleSignal {
    pub bullish_close: bool,
    pub upper_wick_ratio: Option<f64>,
    pub positive_body_ratio: Option<f64>,
    pub above_vwap: Option<bool>,
    pub volume_spike: Option<bool>,
    pub trade_count_spike: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "camelCase")]
pub struct ScoreExplanation {
    pub theme_reasons: Vec<String>,
    pub news_reasons: Vec<String>,
    pub technical_reasons: Vec<String>,
    pub risk_reasons: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
#[serde(rename_all = "camelCase")]
pub struct StructuredReasoning {
    pub reasoning: String,
    pub key_news_points: Vec<String>,
    pub caution_points: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StrategyCheckpoint {
    pub label: String,
    pub source: String,
    pub passed: bool,
    pub note: String,
}
