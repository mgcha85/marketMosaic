use std::path::Path;

use chrono::{DateTime, Duration, NaiveDate, NaiveTime, Utc};
use rusqlite::{params, Connection};

use crate::model::signal::{NewsArticle, SignalCandidate, StockHistoryPoint};
use crate::service::error::AppResult;

pub fn get_history_dates(path: &Path, limit: usize) -> AppResult<Vec<String>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        "SELECT DISTINCT crawl_date FROM stock_history ORDER BY crawl_date DESC LIMIT ?1",
    )?;
    let rows = stmt.query_map([limit as i64], |row| row.get::<_, String>(0))?;
    rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
}

pub fn get_candidates_for_date(path: &Path, date: &str, limit: Option<usize>) -> AppResult<Vec<SignalCandidate>> {
    let conn = Connection::open(path)?;
    let base_sql = r#"
        WITH themed AS (
            SELECT
                sh.code,
                sh.name,
                ts.theme_idx,
                COALESCE(t.name, sh.name) AS theme_name,
                sh.current_price,
                sh.change_rate,
                sh.market_cap,
                sh.volume_index,
                COUNT(*) OVER (PARTITION BY ts.theme_idx) AS theme_stock_count,
                SUM(CASE WHEN COALESCE(sh.change_rate, 0) > 0 THEN 1 ELSE 0 END) OVER (PARTITION BY ts.theme_idx) AS theme_positive_count,
                ROW_NUMBER() OVER (
                    PARTITION BY ts.theme_idx
                    ORDER BY COALESCE(sh.change_rate, -999) DESC, COALESCE(sh.market_cap, 0) DESC, sh.code ASC
                ) AS theme_rank
            FROM stock_history sh
            JOIN theme_stocks ts ON ts.stock_code = sh.code
            LEFT JOIN themes t ON t.theme_idx = ts.theme_idx
            WHERE sh.crawl_date = ?1
        ),
        deduped AS (
            SELECT
                *,
                ROW_NUMBER() OVER (
                    PARTITION BY code
                    ORDER BY
                        theme_rank ASC,
                        CASE
                            WHEN theme_stock_count > 0 THEN (theme_positive_count * 1.0) / theme_stock_count
                            ELSE 0
                        END DESC,
                        COALESCE(change_rate, -999) DESC,
                        COALESCE(market_cap, 0) DESC,
                        theme_idx ASC
                ) AS code_rank
            FROM themed
        )
        SELECT
            code,
            name,
            theme_idx,
            theme_name,
            current_price,
            change_rate,
            market_cap,
            volume_index,
            theme_stock_count,
            theme_rank,
            theme_positive_count
        FROM deduped
        WHERE code_rank = 1
        ORDER BY COALESCE(change_rate, 0) DESC, COALESCE(market_cap, 0) DESC
        "#;
    let sql = if limit.is_some() {
        format!("{base_sql} LIMIT ?2")
    } else {
        base_sql.to_string()
    };
    let mut stmt = conn.prepare(&sql)?;

    let map_row = |row: &rusqlite::Row<'_>| {
        Ok(SignalCandidate {
            code: row.get(0)?,
            name: row.get(1)?,
            theme_idx: row.get(2)?,
            theme_name: row.get(3)?,
            current_price: row.get(4)?,
            change_rate: row.get(5)?,
            market_cap: row.get(6)?,
            volume_index: row.get(7)?,
            theme_stock_count: row.get(8)?,
            theme_rank: row.get(9)?,
            theme_positive_count: row.get(10)?,
        })
    };

    let rows = if let Some(limit) = limit {
        stmt.query_map(params![date, limit as i64], map_row)?
    } else {
        stmt.query_map(params![date], map_row)?
    };

    rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
}

pub fn get_candidate_for_date(path: &Path, date: &str, code: &str) -> AppResult<Option<SignalCandidate>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        r#"
        WITH themed AS (
            SELECT
                sh.code,
                sh.name,
                ts.theme_idx,
                COALESCE(t.name, sh.name) AS theme_name,
                sh.current_price,
                sh.change_rate,
                sh.market_cap,
                sh.volume_index,
                COUNT(*) OVER (PARTITION BY ts.theme_idx) AS theme_stock_count,
                SUM(CASE WHEN COALESCE(sh.change_rate, 0) > 0 THEN 1 ELSE 0 END) OVER (PARTITION BY ts.theme_idx) AS theme_positive_count,
                ROW_NUMBER() OVER (
                    PARTITION BY ts.theme_idx
                    ORDER BY COALESCE(sh.change_rate, -999) DESC, COALESCE(sh.market_cap, 0) DESC, sh.code ASC
                ) AS theme_rank
            FROM stock_history sh
            JOIN theme_stocks ts ON ts.stock_code = sh.code
            LEFT JOIN themes t ON t.theme_idx = ts.theme_idx
            WHERE sh.crawl_date = ?1
        ),
        deduped AS (
            SELECT
                *,
                ROW_NUMBER() OVER (
                    PARTITION BY code
                    ORDER BY
                        theme_rank ASC,
                        CASE
                            WHEN theme_stock_count > 0 THEN (theme_positive_count * 1.0) / theme_stock_count
                            ELSE 0
                        END DESC,
                        COALESCE(change_rate, -999) DESC,
                        COALESCE(market_cap, 0) DESC,
                        theme_idx ASC
                ) AS code_rank
            FROM themed
        )
        SELECT
            code,
            name,
            theme_idx,
            theme_name,
            current_price,
            change_rate,
            market_cap,
            volume_index,
            theme_stock_count,
            theme_rank,
            theme_positive_count
                FROM deduped
        WHERE code = ?2
                    AND code_rank = 1
        LIMIT 1
        "#,
    )?;

    let result = stmt.query_row(params![date, code], |row| {
        Ok(SignalCandidate {
            code: row.get(0)?,
            name: row.get(1)?,
            theme_idx: row.get(2)?,
            theme_name: row.get(3)?,
            current_price: row.get(4)?,
            change_rate: row.get(5)?,
            market_cap: row.get(6)?,
            volume_index: row.get(7)?,
            theme_stock_count: row.get(8)?,
            theme_rank: row.get(9)?,
            theme_positive_count: row.get(10)?,
        })
    });

    Ok(result.ok())
}

pub fn get_recent_prices(path: &Path, code: &str, before_or_equal_date: &str, limit: usize) -> AppResult<Vec<i64>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        r#"
        SELECT current_price, change_rate
        FROM stock_history
        WHERE code = ?1
          AND crawl_date <= ?2
        ORDER BY crawl_date DESC
        LIMIT ?3
        "#,
    )?;
    let rows = stmt.query_map(params![code, before_or_equal_date, limit as i64], |row| {
        Ok((row.get::<_, Option<i64>>(0)?, row.get::<_, Option<f64>>(1)?))
    })?;
    let snapshots = rows.collect::<Result<Vec<_>, _>>()?;

    let concrete_prices = snapshots
        .iter()
        .filter_map(|(price, _)| *price)
        .collect::<Vec<_>>();
    if !concrete_prices.is_empty() {
        return Ok(concrete_prices);
    }

    Ok(build_synthetic_prices(&snapshots, 100_000))
}

pub fn get_next_price(path: &Path, code: &str, after_date: &str) -> AppResult<Option<(String, i64)>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        r#"
        SELECT crawl_date, current_price, change_rate
        FROM stock_history
        WHERE code = ?1
          AND crawl_date > ?2
        ORDER BY crawl_date ASC
        LIMIT 1
        "#,
    )?;

    let result = stmt.query_row(params![code, after_date], |row| {
        let date = row.get::<_, String>(0)?;
        let current_price = row.get::<_, Option<i64>>(1)?;
        let change_rate = row.get::<_, Option<f64>>(2)?;
        Ok((date, current_price.unwrap_or_else(|| synthetic_next_price(change_rate, 100_000))))
    });

    Ok(result.ok())
}

pub fn get_price_on_date(path: &Path, code: &str, date: &str) -> AppResult<Option<i64>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        r#"
        SELECT current_price
        FROM stock_history
        WHERE code = ?1
          AND crawl_date = ?2
        LIMIT 1
        "#,
    )?;

    let result = stmt.query_row(params![code, date], |row| row.get::<_, Option<i64>>(0));
    match result {
        Ok(Some(price)) => Ok(Some(price)),
        Ok(None) => Ok(Some(100_000)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(error) => Err(error.into()),
    }
}

fn build_synthetic_prices(snapshots: &[(Option<i64>, Option<f64>)], base_price: i64) -> Vec<i64> {
    if snapshots.is_empty() {
        return Vec::new();
    }

    let mut prices = Vec::with_capacity(snapshots.len());
    let mut current = base_price as f64;
    prices.push(base_price);

    for index in 1..snapshots.len() {
        let change_rate = normalize_change_rate(snapshots[index - 1].1);
        let denominator = 1.0 + (change_rate / 100.0);
        if denominator.abs() > f64::EPSILON {
            current /= denominator;
        }
        prices.push(current.round() as i64);
    }

    prices
}

fn synthetic_next_price(change_rate: Option<f64>, base_price: i64) -> i64 {
    let multiplier = 1.0 + (normalize_change_rate(change_rate) / 100.0);
    ((base_price as f64) * multiplier.max(0.01)).round() as i64
}

fn normalize_change_rate(change_rate: Option<f64>) -> f64 {
    change_rate.unwrap_or_default().clamp(-95.0, 500.0)
}

pub fn get_stock_history(path: &Path, code: &str, limit: usize) -> AppResult<Vec<StockHistoryPoint>> {
    let conn = Connection::open(path)?;
    let mut stmt = conn.prepare(
        r#"
        SELECT crawl_date, current_price, change_rate, market_cap
        FROM stock_history
        WHERE code = ?1
        ORDER BY crawl_date DESC
        LIMIT ?2
        "#,
    )?;

    let rows = stmt.query_map(params![code, limit as i64], |row| {
        Ok(StockHistoryPoint {
            crawl_date: row.get(0)?,
            current_price: row.get(1)?,
            change_rate: row.get(2)?,
            market_cap: row.get(3)?,
        })
    })?;

    rows.collect::<Result<Vec<_>, _>>().map_err(Into::into)
}

pub async fn search_related_news(
    client: &reqwest::Client,
    meili_url: &str,
    meili_api_key: &str,
    query: &str,
    limit: usize,
    signal_date: Option<&str>,
) -> AppResult<Vec<NewsArticle>> {
    let filter = signal_date.and_then(build_news_filter);

    let response = client
        .post(format!("{}/indexes/articles/search", meili_url.trim_end_matches('/')))
        .bearer_auth(meili_api_key)
        .json(&serde_json::json!({
            "q": query,
            "limit": limit,
            "sort": ["published_at:desc"],
            "filter": filter
        }))
        .send()
        .await?
        .error_for_status()?;

    let payload: serde_json::Value = response.json().await?;
    let hits = payload["hits"].as_array().cloned().unwrap_or_default();

    let articles = hits
        .into_iter()
        .map(|item| NewsArticle {
            title: item["title"].as_str().unwrap_or_default().to_string(),
            published_at: item["published_at"].as_str().unwrap_or_default().to_string(),
            publisher: item["publisher"].as_str().map(ToString::to_string),
            url: item["url"].as_str().map(ToString::to_string),
            age_hours: item["published_at"]
                .as_str()
                .and_then(parse_age_hours),
        })
        .collect();

    Ok(articles)
}

fn parse_age_hours(value: &str) -> Option<i64> {
    let parsed = DateTime::parse_from_rfc3339(value)
        .or_else(|_| DateTime::parse_from_str(value, "%Y-%m-%d %H:%M:%S%#z"))
        .ok()?;
    let now = Utc::now();
    let duration = now.signed_duration_since(parsed.with_timezone(&Utc));
    Some(duration.num_hours().max(0))
}

fn build_news_filter(signal_date: &str) -> Option<String> {
    let date = NaiveDate::parse_from_str(signal_date, "%Y-%m-%d").ok()?;
    let start = date.checked_sub_signed(Duration::days(3))?.and_time(NaiveTime::MIN);
    let cutoff = date.and_time(NaiveTime::from_hms_opt(15, 0, 0)?);
    Some(format!(
        "published_at >= '{}' AND published_at <= '{}'",
        start.format("%Y-%m-%dT%H:%M:%S"),
        cutoff.format("%Y-%m-%dT%H:%M:%S")
    ))
}
