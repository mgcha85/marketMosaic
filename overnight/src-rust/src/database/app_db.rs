use std::fs;
use std::path::Path;

use rusqlite::{params, Connection};

use crate::model::backtest::{
    BacktestCandidateReview, BacktestDayReview, BacktestDetail, BacktestRun, BacktestScoreBucket,
    BacktestStats, BacktestThemeSnapshot, BacktestTrade, EquityPoint,
};
use crate::model::signal::{ScoreExplanation, Signal, SignalCandidate, StructuredReasoning};
use crate::service::error::{AppError, AppResult};
use crate::service::scoring::ScoreBreakdown;

const MIGRATION_SQL: &str = include_str!("../../migrations/0001_init.sql");

pub fn initialize(path: &Path) -> AppResult<()> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)?;
    }

    let conn = Connection::open(path)?;
    conn.execute_batch(MIGRATION_SQL)?;
    ensure_optional_columns(&conn)?;
    Ok(())
}

fn ensure_optional_columns(conn: &Connection) -> AppResult<()> {
    let mut stmt = conn.prepare("PRAGMA table_info(backtest_trades)")?;
    let rows = stmt.query_map([], |row| row.get::<_, String>(1))?;
    let columns = rows.collect::<Result<Vec<_>, _>>()?;
    if !columns.iter().any(|column| column == "explanation_summary") {
        conn.execute(
            "ALTER TABLE backtest_trades ADD COLUMN explanation_summary TEXT",
            [],
        )?;
    }
    if !columns.iter().any(|column| column == "explanation_json") {
        conn.execute(
            "ALTER TABLE backtest_trades ADD COLUMN explanation_json TEXT",
            [],
        )?;
    }

    let mut signal_stmt = conn.prepare("PRAGMA table_info(signals)")?;
    let signal_rows = signal_stmt.query_map([], |row| row.get::<_, String>(1))?;
    let signal_columns = signal_rows.collect::<Result<Vec<_>, _>>()?;
    if !signal_columns
        .iter()
        .any(|column| column == "structured_reasoning")
    {
        conn.execute(
            "ALTER TABLE signals ADD COLUMN structured_reasoning TEXT DEFAULT '{}'",
            [],
        )?;
    }
    if !signal_columns
        .iter()
        .any(|column| column == "explanation_json")
    {
        conn.execute("ALTER TABLE signals ADD COLUMN explanation_json TEXT", [])?;
    }

    conn.execute_batch(
        r#"
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
            candle_signal_json TEXT
        );
        CREATE INDEX IF NOT EXISTS idx_backtest_candidates_run_date ON backtest_candidates(run_id, entry_date);
        "#,
    )?;
    Ok(())
}

pub fn list_signals(path: &Path, date: Option<&str>) -> AppResult<Vec<Signal>> {
    let conn = Connection::open(path)?;
    let sql = if date.is_some() {
        "SELECT id, signal_date, code, name, theme_idx, theme_name, score, score_theme, score_news, score_technical, score_risk, reasoning, entry_price, change_rate, ma5, ma5_support, news_count, news_titles, structured_reasoning, status, created_at FROM signals WHERE signal_date = ?1 ORDER BY score DESC, name ASC"
    } else {
        "SELECT id, signal_date, code, name, theme_idx, theme_name, score, score_theme, score_news, score_technical, score_risk, reasoning, entry_price, change_rate, ma5, ma5_support, news_count, news_titles, structured_reasoning, status, created_at FROM signals ORDER BY signal_date DESC, score DESC, name ASC"
    };

    let mut stmt = conn.prepare(sql)?;

    if let Some(date) = date {
        let rows = stmt.query_map([date], map_signal_row)?;
        rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
    } else {
        let rows = stmt.query_map([], map_signal_row)?;
        rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
    }
}

pub fn delete_signals_for_date(path: &Path, date: &str) -> AppResult<()> {
    let conn = Connection::open(path)?;
    conn.execute("DELETE FROM signals WHERE signal_date = ?1", [date])?;
    Ok(())
}

pub fn upsert_signal(
    path: &Path,
    date: &str,
    candidate: &SignalCandidate,
    score: &ScoreBreakdown,
    reasoning: &str,
    structured_reasoning: &StructuredReasoning,
    explanation: &ScoreExplanation,
    news_titles: &[String],
    news_count: usize,
    status: &str,
) -> AppResult<()> {
    let conn = Connection::open(path)?;
    conn.execute(
        r#"
        INSERT INTO signals (
            signal_date, code, name, theme_idx, theme_name, score,
            score_theme, score_news, score_technical, score_risk,
            reasoning, entry_price, change_rate, ma5, ma5_support,
            news_count, news_titles, explanation_json, structured_reasoning, status
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19, ?20)
        ON CONFLICT(signal_date, code) DO UPDATE SET
            name = excluded.name,
            theme_idx = excluded.theme_idx,
            theme_name = excluded.theme_name,
            score = excluded.score,
            score_theme = excluded.score_theme,
            score_news = excluded.score_news,
            score_technical = excluded.score_technical,
            score_risk = excluded.score_risk,
            reasoning = excluded.reasoning,
            entry_price = excluded.entry_price,
            change_rate = excluded.change_rate,
            ma5 = excluded.ma5,
            ma5_support = excluded.ma5_support,
            news_count = excluded.news_count,
            news_titles = excluded.news_titles,
            explanation_json = excluded.explanation_json,
            structured_reasoning = excluded.structured_reasoning,
            status = excluded.status
        "#,
        params![
            date,
            candidate.code,
            candidate.name,
            candidate.theme_idx,
            candidate.theme_name,
            score.total,
            score.theme,
            score.news,
            score.technical,
            score.risk,
            reasoning,
            candidate.current_price.map(|v| v as f64),
            candidate.change_rate,
            score.ma5,
            if score.ma5_support { 1 } else { 0 },
            news_count as i64,
            serde_json::to_string(news_titles)?,
            serde_json::to_string(explanation)?,
            serde_json::to_string(structured_reasoning)?,
            status,
        ],
    )?;

    Ok(())
}

pub fn replace_backtest_candidates(
    path: &Path,
    run_id: i64,
    entry_date: &str,
    candidates: &[BacktestCandidateReview],
) -> AppResult<()> {
    let conn = Connection::open(path)?;
    conn.execute(
        "DELETE FROM backtest_candidates WHERE run_id = ?1 AND entry_date = ?2",
        params![run_id, entry_date],
    )?;

    for candidate in candidates {
        conn.execute(
            r#"
            INSERT INTO backtest_candidates (
                run_id, entry_date, code, name, theme_name, status, passed_score, was_traded,
                score, score_theme, score_news, score_technical, score_risk, change_rate,
                news_count, ma5_support, reasoning, structured_reasoning_json, explanation_json,
                strategy_checkpoints_json, rejection_reasons_json, exit_date, exit_price, pnl_pct,
                theme_rank, theme_stock_count, theme_positive_count, history_json, related_news_json,
                candle_signal_json
            ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19, ?20, ?21, ?22, ?23, ?24, ?25, ?26, ?27, ?28, ?29, ?30)
            "#,
            params![
                run_id,
                entry_date,
                candidate.code,
                candidate.name,
                candidate.theme_name,
                candidate.status,
                if candidate.passed_score { 1 } else { 0 },
                if candidate.was_traded { 1 } else { 0 },
                candidate.score,
                candidate.score_theme,
                candidate.score_news,
                candidate.score_technical,
                candidate.score_risk,
                candidate.change_rate,
                candidate.news_count,
                if candidate.ma5_support { 1 } else { 0 },
                candidate.reasoning,
                serde_json::to_string(&candidate.structured_reasoning)?,
                serde_json::to_string(&candidate.explanation)?,
                serde_json::to_string(&candidate.strategy_checkpoints)?,
                serde_json::to_string(&candidate.rejection_reasons)?,
                candidate.exit_date,
                candidate.exit_price,
                candidate.pnl_pct,
                candidate.theme_rank,
                candidate.theme_stock_count,
                candidate.theme_positive_count,
                serde_json::to_string(&candidate.history)?,
                serde_json::to_string(&candidate.related_news)?,
                candidate
                    .candle_signal
                    .as_ref()
                    .map(serde_json::to_string)
                    .transpose()?,
            ],
        )?;
    }

    Ok(())
}

pub fn get_backtest_day_review(
    path: &Path,
    run_id: i64,
    date: &str,
) -> AppResult<Option<BacktestDayReview>> {
    let conn = Connection::open(path)?;
    let run = conn
        .query_row(
            "SELECT id, run_at, date_from, date_to, min_score, total_trades, win_count, loss_count, win_rate, avg_return, total_return, max_drawdown, status, created_at FROM backtest_runs WHERE id = ?1",
            [run_id],
            map_backtest_run_row,
        )
        .ok();
    let Some(run) = run else {
        return Ok(None);
    };

    let min_score = run.min_score;

    let mut stmt = conn.prepare(
        r#"
        SELECT code, name, theme_name, status, passed_score, was_traded, score, score_theme, score_news,
               score_technical, score_risk, change_rate, news_count, ma5_support, reasoning,
               structured_reasoning_json, explanation_json, strategy_checkpoints_json, rejection_reasons_json,
               exit_date, exit_price, pnl_pct, theme_rank, theme_stock_count, theme_positive_count,
               history_json, related_news_json, candle_signal_json
        FROM backtest_candidates
        WHERE run_id = ?1 AND entry_date = ?2
        ORDER BY score DESC, name ASC
        "#,
    )?;
    let rows = stmt.query_map(params![run_id, date], |row| {
        let structured_reasoning_json: String = row.get(15)?;
        let explanation_json: String = row.get(16)?;
        let checkpoints_json: String = row.get(17)?;
        let rejection_reasons_json: String = row.get(18)?;
        let history_json: String = row.get(25)?;
        let related_news_json: String = row.get(26)?;
        let candle_signal_json: Option<String> = row.get(27)?;

        Ok(BacktestCandidateReview {
            code: row.get(0)?,
            name: row.get(1)?,
            theme_name: row.get(2)?,
            status: row.get(3)?,
            passed_score: row.get::<_, i64>(4)? == 1,
            was_traded: row.get::<_, i64>(5)? == 1,
            score: row.get(6)?,
            score_theme: row.get(7)?,
            score_news: row.get(8)?,
            score_technical: row.get(9)?,
            score_risk: row.get(10)?,
            change_rate: row.get(11)?,
            news_count: row.get(12)?,
            ma5_support: row.get::<_, i64>(13)? == 1,
            reasoning: row.get(14)?,
            structured_reasoning: serde_json::from_str(&structured_reasoning_json).unwrap_or(
                StructuredReasoning {
                    reasoning: String::new(),
                    key_news_points: vec![],
                    caution_points: vec![],
                },
            ),
            explanation: serde_json::from_str(&explanation_json).unwrap_or_default(),
            strategy_checkpoints: serde_json::from_str(&checkpoints_json).unwrap_or_default(),
            rejection_reasons: serde_json::from_str(&rejection_reasons_json).unwrap_or_default(),
            exit_date: row.get(19)?,
            exit_price: row.get(20)?,
            pnl_pct: row.get(21)?,
            theme_rank: row.get(22)?,
            theme_stock_count: row.get(23)?,
            theme_positive_count: row.get(24)?,
            history: serde_json::from_str(&history_json).unwrap_or_default(),
            related_news: serde_json::from_str(&related_news_json).unwrap_or_default(),
            candle_signal: candle_signal_json.and_then(|json| serde_json::from_str(&json).ok()),
        })
    })?;
    let candidates = rows.collect::<Result<Vec<_>, _>>()?;
    if candidates.is_empty() {
        return Ok(None);
    }

    let themes = build_day_review_themes(&candidates);
    let traded_count = candidates.iter().filter(|item| item.was_traded).count() as i64;
    let pass_count = candidates.iter().filter(|item| item.passed_score).count() as i64;
    let fail_count = candidates.len() as i64 - pass_count;

    Ok(Some(BacktestDayReview {
        run,
        date: date.to_string(),
        min_score,
        traded_count,
        pass_count,
        fail_count,
        themes,
        candidates,
    }))
}

fn build_day_review_themes(candidates: &[BacktestCandidateReview]) -> Vec<BacktestThemeSnapshot> {
    let mut grouped = std::collections::BTreeMap::<String, Vec<&BacktestCandidateReview>>::new();
    for candidate in candidates {
        grouped
            .entry(
                candidate
                    .theme_name
                    .clone()
                    .unwrap_or_else(|| "미분류".to_string()),
            )
            .or_default()
            .push(candidate);
    }

    let mut themes = grouped
        .into_iter()
        .map(|(theme_name, items)| {
            let candidate_count = items.len() as i64;
            let pass_count = items.iter().filter(|item| item.passed_score).count() as i64;
            let average_score = if candidate_count > 0 {
                items.iter().map(|item| item.score).sum::<f64>() / candidate_count as f64
            } else {
                0.0
            };
            let breadth_note = items.first().and_then(|item| {
                match (item.theme_positive_count, item.theme_stock_count) {
                    (Some(positive), Some(total)) if total > 0 => {
                        Some(format!("상승 종목 {positive}/{total}"))
                    }
                    _ => None,
                }
            });

            BacktestThemeSnapshot {
                theme_name,
                candidate_count,
                pass_count,
                average_score,
                breadth_note,
                lead_stock: items.first().map(|item| item.name.clone()),
            }
        })
        .collect::<Vec<_>>();
    themes.sort_by(|left, right| {
        right
            .average_score
            .partial_cmp(&left.average_score)
            .unwrap_or(std::cmp::Ordering::Equal)
    });
    themes
}

pub fn insert_backtest_run(path: &Path, run: &BacktestRun) -> AppResult<i64> {
    let conn = Connection::open(path)?;
    conn.execute(
        r#"
        INSERT INTO backtest_runs (
            run_at, date_from, date_to, min_score, total_trades,
            win_count, loss_count, win_rate, avg_return, total_return,
            max_drawdown, status, created_at
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)
        "#,
        params![
            run.run_at,
            run.date_from,
            run.date_to,
            run.min_score,
            run.total_trades,
            run.win_count,
            run.loss_count,
            run.win_rate,
            run.avg_return,
            run.total_return,
            run.max_drawdown,
            run.status,
            run.created_at,
        ],
    )?;
    Ok(conn.last_insert_rowid())
}

pub fn insert_backtest_trade(path: &Path, trade: &BacktestTrade) -> AppResult<()> {
    let conn = Connection::open(path)?;
    conn.execute(
        r#"
        INSERT INTO backtest_trades (
            run_id, code, name, theme_name, entry_date, exit_date,
            entry_price, exit_price, pnl_pct, score, score_theme,
            score_news, score_technical, news_count, ma5_support, explanation_summary, explanation_json
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17)
        "#,
        params![
            trade.run_id,
            trade.code,
            trade.name,
            trade.theme_name,
            trade.entry_date,
            trade.exit_date,
            trade.entry_price,
            trade.exit_price,
            trade.pnl_pct,
            trade.score,
            trade.score_theme,
            trade.score_news,
            trade.score_technical,
            trade.news_count,
            trade.ma5_support.map(|v| if v { 1 } else { 0 }),
            trade.explanation_summary,
            trade.explanation.as_ref().map(|value| serde_json::to_string(value)).transpose()?,
        ],
    )?;
    Ok(())
}

pub fn list_backtest_runs(path: &Path) -> AppResult<Vec<BacktestRun>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        "SELECT id, run_at, date_from, date_to, min_score, total_trades, win_count, loss_count, win_rate, avg_return, total_return, max_drawdown, status, created_at FROM backtest_runs ORDER BY created_at DESC",
    )?;
    let rows = stmt.query_map([], map_backtest_run_row)?;
    rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
}

pub fn get_backtest_detail(path: &Path, run_id: i64) -> AppResult<BacktestDetail> {
    let conn = Connection::open(path)?;

    let run = conn.query_row(
        "SELECT id, run_at, date_from, date_to, min_score, total_trades, win_count, loss_count, win_rate, avg_return, total_return, max_drawdown, status, created_at FROM backtest_runs WHERE id = ?1",
        [run_id],
        map_backtest_run_row,
    ).map_err(|_| AppError::NotFound(format!("backtest run {} not found", run_id)))?;

    let mut stmt = conn.prepare(
        "SELECT id, run_id, code, name, theme_name, entry_date, exit_date, entry_price, exit_price, pnl_pct, score, score_theme, score_news, score_technical, news_count, ma5_support, explanation_summary, explanation_json FROM backtest_trades WHERE run_id = ?1 ORDER BY entry_date ASC, code ASC",
    )?;
    let rows = stmt.query_map([run_id], map_backtest_trade_row)?;
    let trades = rows.collect::<Result<Vec<_>, _>>()?;
    let equity_curve = build_equity_curve(&trades);
    let candidate_dates = load_backtest_candidate_dates(&conn, run_id)?;
    let score_buckets = load_backtest_score_buckets(&conn, run_id)?;

    Ok(BacktestDetail {
        run,
        trades,
        equity_curve,
        candidate_dates,
        score_buckets,
    })
}

fn load_backtest_candidate_dates(conn: &Connection, run_id: i64) -> AppResult<Vec<String>> {
    let mut stmt = conn.prepare(
        "SELECT DISTINCT entry_date FROM backtest_candidates WHERE run_id = ?1 ORDER BY entry_date DESC",
    )?;
    let rows = stmt.query_map([run_id], |row| row.get::<_, String>(0))?;
    rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
}

fn load_backtest_score_buckets(conn: &Connection, run_id: i64) -> AppResult<Vec<BacktestScoreBucket>> {
    let mut stmt = conn.prepare(
        r#"
        SELECT score, was_traded, pnl_pct
        FROM backtest_candidates
        WHERE run_id = ?1
        ORDER BY score DESC
        "#,
    )?;
    let rows = stmt.query_map([run_id], |row| {
        Ok((
            row.get::<_, f64>(0)?,
            row.get::<_, i64>(1)? == 1,
            row.get::<_, Option<f64>>(2)?,
        ))
    })?;
    let items = rows.collect::<Result<Vec<_>, _>>()?;

    let mut buckets = std::collections::BTreeMap::<i64, Vec<(f64, bool, Option<f64>)>>::new();
    for item in items {
        let bucket_key = item.0.floor() as i64;
        buckets.entry(bucket_key).or_default().push(item);
    }

    let mut summaries = buckets
        .into_iter()
        .rev()
        .map(|(bucket_key, items)| {
            let candidate_count = items.len() as i64;
            let outcome_items = items
                .iter()
                .filter_map(|(score, traded, pnl_pct)| pnl_pct.map(|pnl| (*score, *traded, pnl)))
                .collect::<Vec<_>>();
            let outcome_count = outcome_items.len() as i64;
            let traded_count = items.iter().filter(|(_, traded, _)| *traded).count() as i64;
            let win_count = outcome_items.iter().filter(|(_, _, pnl)| *pnl > 0.0).count() as i64;
            let avg_score = if candidate_count > 0 {
                items.iter().map(|(score, _, _)| *score).sum::<f64>() / candidate_count as f64
            } else {
                0.0
            };
            let avg_return = if outcome_count > 0 {
                outcome_items.iter().map(|(_, _, pnl)| *pnl).sum::<f64>() / outcome_count as f64
            } else {
                0.0
            };
            let win_rate = if outcome_count > 0 {
                (win_count as f64 / outcome_count as f64) * 100.0
            } else {
                0.0
            };
            let min_score = bucket_key as f64;
            let max_score = min_score + 0.99;

            BacktestScoreBucket {
                label: format!("{min_score:.0}점대"),
                min_score,
                max_score,
                candidate_count,
                outcome_count,
                traded_count,
                win_count,
                win_rate,
                avg_score,
                avg_return,
            }
        })
        .collect::<Vec<_>>();

    summaries.sort_by(|left, right| right.min_score.partial_cmp(&left.min_score).unwrap_or(std::cmp::Ordering::Equal));
    Ok(summaries)
}

pub fn get_backtest_stats(path: &Path) -> AppResult<BacktestStats> {
    let conn = Connection::open(path)?;
    let (total_runs, total_trades, avg_win_rate, avg_return) = conn.query_row(
        "SELECT COUNT(*), COALESCE(SUM(total_trades), 0), COALESCE(AVG(win_rate), 0), COALESCE(AVG(avg_return), 0) FROM backtest_runs",
        [],
        |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?, row.get(3)?)),
    )?;

    let best_run = conn
        .query_row(
            "SELECT id, run_at, date_from, date_to, min_score, total_trades, win_count, loss_count, win_rate, avg_return, total_return, max_drawdown, status, created_at FROM backtest_runs ORDER BY total_return DESC LIMIT 1",
            [],
            map_backtest_run_row,
        )
        .ok();

    Ok(BacktestStats {
        total_runs,
        total_trades,
        avg_win_rate,
        avg_return,
        best_run,
    })
}

fn build_equity_curve(trades: &[BacktestTrade]) -> Vec<EquityPoint> {
    let mut equity = 1.0;
    trades
        .iter()
        .map(|trade| {
            equity *= 1.0 + (trade.pnl_pct / 100.0);
            EquityPoint {
                date: trade.exit_date.clone(),
                equity,
                return_pct: (equity - 1.0) * 100.0,
            }
        })
        .collect()
}

fn map_signal_row(row: &rusqlite::Row<'_>) -> rusqlite::Result<Signal> {
    let news_titles_json: String = row.get(17)?;
    let structured_reasoning_json: String = row.get(18)?;
    let news_titles = serde_json::from_str(&news_titles_json).unwrap_or_default();
    let structured_reasoning =
        serde_json::from_str(&structured_reasoning_json).unwrap_or(StructuredReasoning {
            reasoning: row.get::<_, String>(11).unwrap_or_default(),
            key_news_points: vec![],
            caution_points: vec![],
        });
    Ok(Signal {
        id: row.get(0)?,
        signal_date: row.get(1)?,
        code: row.get(2)?,
        name: row.get(3)?,
        theme_idx: row.get(4)?,
        theme_name: row.get(5)?,
        score: row.get(6)?,
        score_theme: row.get(7)?,
        score_news: row.get(8)?,
        score_technical: row.get(9)?,
        score_risk: row.get(10)?,
        reasoning: row.get(11)?,
        entry_price: row.get(12)?,
        change_rate: row.get(13)?,
        ma5: row.get(14)?,
        ma5_support: row.get::<_, i64>(15)? == 1,
        news_count: row.get(16)?,
        news_titles,
        structured_reasoning,
        status: row.get(19)?,
        created_at: row.get(20)?,
    })
}

fn map_backtest_run_row(row: &rusqlite::Row<'_>) -> rusqlite::Result<BacktestRun> {
    Ok(BacktestRun {
        id: row.get(0)?,
        run_at: row.get(1)?,
        date_from: row.get(2)?,
        date_to: row.get(3)?,
        min_score: row.get(4)?,
        total_trades: row.get(5)?,
        win_count: row.get(6)?,
        loss_count: row.get(7)?,
        win_rate: row.get(8)?,
        avg_return: row.get(9)?,
        total_return: row.get(10)?,
        max_drawdown: row.get(11)?,
        status: row.get(12)?,
        created_at: row.get(13)?,
    })
}

fn map_backtest_trade_row(row: &rusqlite::Row<'_>) -> rusqlite::Result<BacktestTrade> {
    Ok(BacktestTrade {
        id: row.get(0)?,
        run_id: row.get(1)?,
        code: row.get(2)?,
        name: row.get(3)?,
        theme_name: row.get(4)?,
        entry_date: row.get(5)?,
        exit_date: row.get(6)?,
        entry_price: row.get(7)?,
        exit_price: row.get(8)?,
        pnl_pct: row.get(9)?,
        score: row.get(10)?,
        score_theme: row.get(11)?,
        score_news: row.get(12)?,
        score_technical: row.get(13)?,
        news_count: row.get(14)?,
        ma5_support: row.get::<_, Option<i64>>(15)?.map(|value| value == 1),
        explanation_summary: row.get(16)?,
        explanation: row
            .get::<_, Option<String>>(17)?
            .and_then(|json| serde_json::from_str(&json).ok()),
    })
}
