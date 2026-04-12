use std::path::Path;

use duckdb::Connection;

use crate::model::signal::CandleSignal;
use crate::service::error::AppResult;

pub fn get_latest_candle_signal(
    candles_dir: &Path,
    code: &str,
    market: &str,
) -> AppResult<Option<CandleSignal>> {
    if !candles_dir.exists() {
        return Ok(None);
    }

    let glob = format!("{}/**/*.parquet", candles_dir.display());
    let conn = Connection::open_in_memory()?;
    let sql = format!(
        r#"
        SELECT
            symbol,
            open,
            high,
            low,
            close,
            volume,
            timestamp,
            trade_count,
            vwap
        FROM read_parquet('{glob}', hive_partitioning = true)
        WHERE market = ?
          AND symbol = ?
        ORDER BY timestamp DESC
        LIMIT 5
        "#
    );

    let mut stmt = match conn.prepare(&sql) {
        Ok(stmt) => stmt,
        Err(_) => return Ok(None),
    };

    let mut rows = match stmt.query([market, code]) {
        Ok(rows) => rows,
        Err(_) => return Ok(None),
    };

    let mut candles = Vec::new();
    while let Some(row) = rows.next()? {
        let open: f64 = row.get::<_, Option<f64>>(1)?.unwrap_or_default();
        let high: f64 = row.get::<_, Option<f64>>(2)?.unwrap_or_default();
        let low: f64 = row.get::<_, Option<f64>>(3)?.unwrap_or_default();
        let close: f64 = row.get::<_, Option<f64>>(4)?.unwrap_or_default();
        let volume: f64 = row.get::<_, Option<f64>>(5)?.unwrap_or_default();
        let trade_count: f64 = row.get::<_, Option<f64>>(7)?.unwrap_or_default();
        let vwap: f64 = row.get::<_, Option<f64>>(8)?.unwrap_or_default();
        candles.push((open, high, low, close, volume, trade_count, vwap));
    }

    let Some((open, high, low, close, volume, trade_count, vwap)) = candles.first().copied() else {
        return Ok(None);
    };

    let range = (high - low).abs();
    let upper_wick_ratio = if range > 0.0 {
        Some(((high - close.max(open)).max(0.0)) / range)
    } else {
        None
    };
    let positive_body_ratio = if range > 0.0 {
        Some(((close - open).max(0.0)) / range)
    } else {
        None
    };

    let avg_volume = average(candles.iter().skip(1).map(|item| item.4));
    let avg_trade_count = average(candles.iter().skip(1).map(|item| item.5));

    Ok(Some(CandleSignal {
        bullish_close: close >= open && close > 0.0,
        upper_wick_ratio,
        positive_body_ratio,
        above_vwap: Some(vwap > 0.0 && close >= vwap),
        volume_spike: avg_volume.map(|avg| avg > 0.0 && volume >= avg * 1.2),
        trade_count_spike: avg_trade_count.map(|avg| avg > 0.0 && trade_count >= avg * 1.2),
    }))
}

fn average<I>(values: I) -> Option<f64>
where
    I: Iterator<Item = f64>,
{
    let mut count = 0.0;
    let mut sum = 0.0;
    for value in values {
        sum += value;
        count += 1.0;
    }
    if count > 0.0 {
        Some(sum / count)
    } else {
        None
    }
}
