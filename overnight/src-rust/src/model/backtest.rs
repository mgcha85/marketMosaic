use crate::model::signal::{
    CandleSignal, NewsArticle, ScoreExplanation, StockHistoryPoint, StrategyCheckpoint,
    StructuredReasoning,
};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestRun {
    pub id: i64,
    pub run_at: String,
    pub date_from: String,
    pub date_to: String,
    pub min_score: f64,
    pub total_trades: i64,
    pub win_count: i64,
    pub loss_count: i64,
    pub win_rate: f64,
    pub avg_return: f64,
    pub total_return: f64,
    pub max_drawdown: f64,
    pub status: String,
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestTrade {
    pub id: i64,
    pub run_id: i64,
    pub code: String,
    pub name: String,
    pub theme_name: Option<String>,
    pub entry_date: String,
    pub exit_date: String,
    pub entry_price: f64,
    pub exit_price: f64,
    pub pnl_pct: f64,
    pub score: Option<f64>,
    pub score_theme: Option<f64>,
    pub score_news: Option<f64>,
    pub score_technical: Option<f64>,
    pub news_count: Option<i64>,
    pub ma5_support: Option<bool>,
    pub explanation_summary: Option<String>,
    pub explanation: Option<ScoreExplanation>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct EquityPoint {
    pub date: String,
    pub equity: f64,
    pub return_pct: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestScoreBucket {
    pub label: String,
    pub min_score: f64,
    pub max_score: f64,
    pub candidate_count: i64,
    pub outcome_count: i64,
    pub traded_count: i64,
    pub win_count: i64,
    pub win_rate: f64,
    pub avg_score: f64,
    pub avg_return: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestDetail {
    pub run: BacktestRun,
    pub trades: Vec<BacktestTrade>,
    pub equity_curve: Vec<EquityPoint>,
    pub candidate_dates: Vec<String>,
    pub score_buckets: Vec<BacktestScoreBucket>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestThemeSnapshot {
    pub theme_name: String,
    pub candidate_count: i64,
    pub pass_count: i64,
    pub average_score: f64,
    pub breadth_note: Option<String>,
    pub lead_stock: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestCandidateReview {
    pub code: String,
    pub name: String,
    pub theme_name: Option<String>,
    pub status: String,
    pub passed_score: bool,
    pub was_traded: bool,
    pub score: f64,
    pub score_theme: f64,
    pub score_news: f64,
    pub score_technical: f64,
    pub score_risk: f64,
    pub change_rate: Option<f64>,
    pub news_count: i64,
    pub ma5_support: bool,
    pub reasoning: String,
    pub structured_reasoning: StructuredReasoning,
    pub explanation: ScoreExplanation,
    pub strategy_checkpoints: Vec<StrategyCheckpoint>,
    pub rejection_reasons: Vec<String>,
    pub exit_date: Option<String>,
    pub exit_price: Option<f64>,
    pub pnl_pct: Option<f64>,
    pub theme_rank: Option<i64>,
    pub theme_stock_count: Option<i64>,
    pub theme_positive_count: Option<i64>,
    pub history: Vec<StockHistoryPoint>,
    pub related_news: Vec<NewsArticle>,
    pub candle_signal: Option<CandleSignal>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestDayReview {
    pub run: BacktestRun,
    pub date: String,
    pub min_score: f64,
    pub traded_count: i64,
    pub pass_count: i64,
    pub fail_count: i64,
    pub themes: Vec<BacktestThemeSnapshot>,
    pub candidates: Vec<BacktestCandidateReview>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestStats {
    pub total_runs: i64,
    pub total_trades: i64,
    pub avg_win_rate: f64,
    pub avg_return: f64,
    pub best_run: Option<BacktestRun>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RunBacktestResponse {
    pub run: BacktestRun,
    pub trades: Vec<BacktestTrade>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestJobStarted {
    pub job_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct BacktestJobStatus {
    pub job_id: String,
    pub status: String,
    pub date_from: String,
    pub date_to: String,
    pub min_score: f64,
    pub completed_steps: i64,
    pub total_steps: i64,
    pub current_date: Option<String>,
    pub run_id: Option<i64>,
    pub error: Option<String>,
    pub started_at: String,
    pub finished_at: Option<String>,
}
