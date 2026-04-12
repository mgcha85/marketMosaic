use chrono::Utc;
use std::collections::HashMap;

use crate::database::{app_db, candle_db, judal_db};
use crate::model::backtest::{
    BacktestCandidateReview, BacktestDayReview, BacktestDetail, BacktestRun, BacktestThemeSnapshot,
    BacktestTrade, RunBacktestResponse,
};
use crate::model::signal::{
    CandleSignal, ScoreExplanation, Signal, SignalCandidate, StrategyCheckpoint, StructuredReasoning,
};
use crate::service::app_state::AppState;
use crate::service::error::{AppError, AppResult};
use crate::service::scoring::calculate_score;

fn summarize_explanation(explanation: &crate::model::signal::ScoreExplanation) -> Option<String> {
    let mut lines = Vec::new();
    if let Some(item) = explanation.theme_reasons.first() {
        lines.push(format!("테마: {item}"));
    }
    if let Some(item) = explanation.news_reasons.first() {
        lines.push(format!("뉴스: {item}"));
    }
    if let Some(item) = explanation.technical_reasons.first() {
        lines.push(format!("기술: {item}"));
    }
    if let Some(item) = explanation.risk_reasons.first() {
        lines.push(format!("리스크: {item}"));
    }
    if lines.is_empty() {
        None
    } else {
        Some(lines.join(" | "))
    }
}

fn first_non_empty(values: &[String], fallback: impl Into<String>) -> String {
    values
        .iter()
        .find(|value| !value.trim().is_empty())
        .cloned()
        .unwrap_or_else(|| fallback.into())
}

fn configured_candidate_limit() -> Option<usize> {
    std::env::var("BACKTEST_MAX_CANDIDATES_PER_DAY")
        .ok()
        .and_then(|raw| raw.parse::<usize>().ok())
        .filter(|value| *value > 0)
        .or(Some(200))
}

fn build_strategy_checkpoints(
    signal: &Signal,
    explanation: &ScoreExplanation,
    candle_signal: Option<&CandleSignal>,
) -> Vec<StrategyCheckpoint> {
    let structured_reasoning = &signal.structured_reasoning;
    let candle_signal = candle_signal.cloned().unwrap_or_default();

    vec![
        StrategyCheckpoint {
            label: "A급 점수 기준".to_string(),
            source: "전략 3·5".to_string(),
            passed: signal.score >= 8.0,
            note: format!("총점 {:.1} / 10", signal.score),
        },
        StrategyCheckpoint {
            label: "Judal 주도 테마".to_string(),
            source: "전략 1·3·6".to_string(),
            passed: !explanation.theme_reasons.is_empty() || signal.theme_name.is_some(),
            note: first_non_empty(
                &explanation.theme_reasons,
                signal.theme_name.clone().unwrap_or_else(|| "테마 정보 없음".to_string()),
            ),
        },
        StrategyCheckpoint {
            label: "재료 명분 확보".to_string(),
            source: "전략 3·5·6".to_string(),
            passed: signal.news_count > 0 || !structured_reasoning.key_news_points.is_empty(),
            note: first_non_empty(
                &structured_reasoning.key_news_points,
                first_non_empty(
                    &explanation.news_reasons,
                    format!("관련 뉴스 {}건", signal.news_count),
                ),
            ),
        },
        StrategyCheckpoint {
            label: "지지·방어 확인".to_string(),
            source: "전략 2·4".to_string(),
            passed: signal.ma5_support
                || candle_signal.above_vwap == Some(true)
                || candle_signal.bullish_close,
            note: first_non_empty(
                &explanation.technical_reasons,
                if signal.ma5_support {
                    "현재가가 MA5 위에서 지지".to_string()
                } else {
                    "캔들 방어 신호 확인 필요".to_string()
                },
            ),
        },
        StrategyCheckpoint {
            label: "과열 리스크 점검".to_string(),
            source: "전략 1·5".to_string(),
            passed: signal.score_risk.unwrap_or_default() >= 1.0,
            note: first_non_empty(
                &explanation.risk_reasons,
                format!("리스크 점수 {:.1}", signal.score_risk.unwrap_or_default()),
            ),
        },
        StrategyCheckpoint {
            label: "종합 의견 정리".to_string(),
            source: "전략 전체 요약".to_string(),
            passed: !structured_reasoning.reasoning.trim().is_empty() || !signal.reasoning.trim().is_empty(),
            note: if !structured_reasoning.reasoning.trim().is_empty() {
                structured_reasoning.reasoning.clone()
            } else {
                signal.reasoning.clone()
            },
        },
    ]
}

fn build_rejection_reasons(min_score: f64, total_score: f64, explanation: &ScoreExplanation) -> Vec<String> {
    let mut reasons = Vec::new();
    if total_score < min_score {
        reasons.push(format!("총점 {:.1}점으로 기준 {:.1}점 미달", total_score, min_score));
    }
    if let Some(reason) = explanation.news_reasons.first() {
        reasons.push(format!("뉴스 판단: {reason}"));
    }
    if let Some(reason) = explanation.technical_reasons.first() {
        reasons.push(format!("차트 판단: {reason}"));
    }
    if let Some(reason) = explanation.risk_reasons.first() {
        reasons.push(format!("리스크 판단: {reason}"));
    }
    reasons
}

fn build_signal_from_candidate(
    date: &str,
    candidate: &SignalCandidate,
    total_score: f64,
    theme_score: f64,
    news_score: f64,
    technical_score: f64,
    risk_score: f64,
    reasoning: String,
    structured_reasoning: StructuredReasoning,
    news_titles: Vec<String>,
    news_count: i64,
    ma5: Option<f64>,
    ma5_support: bool,
) -> Signal {
    Signal {
        id: 0,
        signal_date: date.to_string(),
        code: candidate.code.clone(),
        name: candidate.name.clone(),
        theme_idx: Some(candidate.theme_idx),
        theme_name: Some(candidate.theme_name.clone()),
        score: total_score,
        score_theme: Some(theme_score),
        score_news: Some(news_score),
        score_technical: Some(technical_score),
        score_risk: Some(risk_score),
        reasoning,
        entry_price: candidate.current_price.map(|value| value as f64),
        change_rate: candidate.change_rate,
        ma5,
        ma5_support,
        news_count,
        news_titles,
        structured_reasoning,
        status: "review".to_string(),
        created_at: Utc::now().to_rfc3339(),
    }
}

pub fn get_backtest_detail(state: &AppState, run_id: i64) -> AppResult<BacktestDetail> {
    let mut detail = app_db::get_backtest_detail(&state.app_db_path, run_id)?;

    if detail.candidate_dates.is_empty() {
        let history_dates = judal_db::get_history_dates(&state.judal_db_path, 365)?;
        detail.candidate_dates = history_dates
            .into_iter()
            .filter(|date| date.as_str() >= detail.run.date_from.as_str() && date.as_str() <= detail.run.date_to.as_str())
            .collect();
    }

    Ok(detail)
}

pub async fn build_day_review(state: &AppState, run_id: i64, date: &str) -> AppResult<BacktestDayReview> {
    let detail = app_db::get_backtest_detail(&state.app_db_path, run_id)?;
    let run = detail.run;

    if date < run.date_from.as_str() || date > run.date_to.as_str() {
        return Err(AppError::Validation(format!(
            "date {date} is outside run range {} ~ {}",
            run.date_from, run.date_to
        )));
    }

    let candidate_limit = configured_candidate_limit();
    let candidates = judal_db::get_candidates_for_date(&state.judal_db_path, date, candidate_limit)?;
    let trade_map: HashMap<String, BacktestTrade> = detail
        .trades
        .into_iter()
        .filter(|trade| trade.entry_date == date)
        .map(|trade| (trade.code.clone(), trade))
        .collect();

    let mut reviews = Vec::new();
    let enable_backtest_news = std::env::var("ENABLE_BACKTEST_NEWS")
        .ok()
        .map(|value| matches!(value.as_str(), "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(true);
    let mut skip_news_lookup = false;

    for candidate in candidates {
        let recent_prices = judal_db::get_recent_prices(&state.judal_db_path, &candidate.code, date, 5)?;
        let related_news = if !enable_backtest_news || skip_news_lookup {
            Vec::new()
        } else {
            match judal_db::search_related_news(
                &state.http_client,
                &state.meili_url,
                &state.meili_api_key,
                &candidate.name,
                5,
                Some(date),
            )
            .await
            {
                Ok(items) => items,
                Err(_) => {
                    skip_news_lookup = true;
                    Vec::new()
                }
            }
        };
        let history = judal_db::get_stock_history(&state.judal_db_path, &candidate.code, 10)?;
        let candle_signal = candle_db::get_latest_candle_signal(&state.candles_dir, &candidate.code, "KR")
            .ok()
            .flatten();
        let score = calculate_score(&candidate, &recent_prices, &related_news, candle_signal.as_ref());
        let news_titles = related_news
            .iter()
            .take(3)
            .map(|article| article.title.clone())
            .collect::<Vec<_>>();
        let structured_reasoning = crate::agent::reasoning::build_reasoning_fast(&candidate, &score, &news_titles);
        let signal = build_signal_from_candidate(
            date,
            &candidate,
            score.total,
            score.theme,
            score.news,
            score.technical,
            score.risk,
            structured_reasoning.reasoning.clone(),
            structured_reasoning.clone(),
            news_titles.clone(),
            related_news.len() as i64,
            score.ma5,
            score.ma5_support,
        );
        let trade = trade_map.get(&candidate.code);
        let passed_score = score.total >= run.min_score;
        let rejection_reasons = if passed_score {
            Vec::new()
        } else {
            build_rejection_reasons(run.min_score, score.total, &score.explanation)
        };
        let strategy_checkpoints = build_strategy_checkpoints(&signal, &score.explanation, candle_signal.as_ref());

        reviews.push(BacktestCandidateReview {
            code: candidate.code.clone(),
            name: candidate.name.clone(),
            theme_name: Some(candidate.theme_name.clone()),
            status: if trade.is_some() {
                "pass".to_string()
            } else if passed_score {
                "candidate".to_string()
            } else {
                "fail".to_string()
            },
            passed_score,
            was_traded: trade.is_some(),
            score: score.total,
            score_theme: score.theme,
            score_news: score.news,
            score_technical: score.technical,
            score_risk: score.risk,
            change_rate: candidate.change_rate,
            news_count: related_news.len() as i64,
            ma5_support: score.ma5_support,
            reasoning: signal.reasoning.clone(),
            structured_reasoning,
            explanation: score.explanation.clone(),
            strategy_checkpoints,
            rejection_reasons,
            exit_date: trade.map(|item| item.exit_date.clone()),
            exit_price: trade.map(|item| item.exit_price),
            pnl_pct: trade.map(|item| item.pnl_pct),
            theme_rank: candidate.theme_rank,
            theme_stock_count: candidate.theme_stock_count,
            theme_positive_count: candidate.theme_positive_count,
            history,
            related_news,
            candle_signal,
        });
    }

    reviews.sort_by(|left, right| {
        right
            .score
            .partial_cmp(&left.score)
            .unwrap_or(std::cmp::Ordering::Equal)
            .then_with(|| left.name.cmp(&right.name))
    });

    let mut theme_map: HashMap<String, Vec<&BacktestCandidateReview>> = HashMap::new();
    for review in &reviews {
        let key = review.theme_name.clone().unwrap_or_else(|| "미분류".to_string());
        theme_map.entry(key).or_default().push(review);
    }

    let mut themes = theme_map
        .into_iter()
        .map(|(theme_name, items)| {
            let candidate_count = items.len() as i64;
            let pass_count = items.iter().filter(|item| item.passed_score).count() as i64;
            let average_score = if candidate_count > 0 {
                items.iter().map(|item| item.score).sum::<f64>() / candidate_count as f64
            } else {
                0.0
            };
            let breadth_note = items.first().and_then(|item| match (item.theme_positive_count, item.theme_stock_count) {
                (Some(positive), Some(total)) if total > 0 => Some(format!("상승 종목 {positive}/{total}")),
                _ => None,
            });
            let lead_stock = items.first().map(|item| item.name.clone());

            BacktestThemeSnapshot {
                theme_name,
                candidate_count,
                pass_count,
                average_score,
                breadth_note,
                lead_stock,
            }
        })
        .collect::<Vec<_>>();
    themes.sort_by(|left, right| {
        right
            .average_score
            .partial_cmp(&left.average_score)
            .unwrap_or(std::cmp::Ordering::Equal)
    });

    let traded_count = reviews.iter().filter(|item| item.was_traded).count() as i64;
    let pass_count = reviews.iter().filter(|item| item.passed_score).count() as i64;
    let fail_count = reviews.len() as i64 - pass_count;

    let min_score = run.min_score;

    Ok(BacktestDayReview {
        run,
        date: date.to_string(),
        min_score,
        traded_count,
        pass_count,
        fail_count,
        themes,
        candidates: reviews,
    })
}

pub async fn run_with_progress<F>(
    state: &AppState,
    date_from: &str,
    date_to: &str,
    min_score: f64,
    mut on_progress: F,
) -> AppResult<RunBacktestResponse>
where
    F: FnMut(i64, i64, Option<&str>) + Send,
{
    let dates = judal_db::get_history_dates(&state.judal_db_path, 365)?;
    let mut filtered_dates: Vec<String> = dates
        .into_iter()
        .filter(|date| date.as_str() >= date_from && date.as_str() <= date_to)
        .collect();
    filtered_dates.sort();

    let run_timestamp = Utc::now().to_rfc3339();
    let mut trades = Vec::new();
    let mut daily_candidate_reviews: Vec<(String, Vec<BacktestCandidateReview>)> = Vec::new();
    let enable_backtest_news = std::env::var("ENABLE_BACKTEST_NEWS")
        .ok()
        .map(|value| matches!(value.as_str(), "1" | "true" | "TRUE" | "yes" | "YES"))
        .unwrap_or(true);
    let mut skip_news_lookup = false;
    let total_steps = filtered_dates.len() as i64;
    let candidate_limit = configured_candidate_limit();

    for (index, date) in filtered_dates.iter().enumerate() {
        on_progress(index as i64, total_steps, Some(date));
    if filtered_dates.is_empty() {
        return Err(AppError::Validation(format!(
            "no history dates available in range [{date_from}, {date_to}]"
        )));
    }
        let candidates = judal_db::get_candidates_for_date(&state.judal_db_path, date, candidate_limit)?;
        let mut reviews_for_date = Vec::new();

        for candidate in candidates {
            let recent_prices = judal_db::get_recent_prices(&state.judal_db_path, &candidate.code, date, 5)?;
            let news = if !enable_backtest_news || skip_news_lookup {
                Vec::new()
            } else {
                match judal_db::search_related_news(
                    &state.http_client,
                    &state.meili_url,
                    &state.meili_api_key,
                    &candidate.name,
                    5,
                    Some(date),
                )
                .await
                {
                    Ok(items) => items,
                    Err(_) => {
                        skip_news_lookup = true;
                        Vec::new()
                    }
                }
            };
            let candle_signal = candle_db::get_latest_candle_signal(&state.candles_dir, &candidate.code, "KR").ok().flatten();
            let score = calculate_score(&candidate, &recent_prices, &news, candle_signal.as_ref());
            let news_count = news.len() as i64;
            let news_titles = news
                .iter()
                .take(3)
                .map(|article| article.title.clone())
                .collect::<Vec<_>>();
            let structured_reasoning = crate::agent::reasoning::build_reasoning_fast(&candidate, &score, &news_titles);
            let signal = build_signal_from_candidate(
                date,
                &candidate,
                score.total,
                score.theme,
                score.news,
                score.technical,
                score.risk,
                structured_reasoning.reasoning.clone(),
                structured_reasoning.clone(),
                news_titles,
                news_count,
                score.ma5,
                score.ma5_support,
            );
            let history = judal_db::get_stock_history(&state.judal_db_path, &candidate.code, 10)?;
            let passed_score = score.total >= min_score;
            let (exit_date, exit_price, pnl_pct) = if let Some((exit_date, exit_price)) =
                judal_db::get_next_price(&state.judal_db_path, &candidate.code, date)?
            {
                let entry_price = candidate
                    .current_price
                    .or(judal_db::get_price_on_date(&state.judal_db_path, &candidate.code, date)?)
                    .unwrap_or(100_000) as f64;
                let exit_price_f64 = exit_price as f64;
                let pnl_pct = if entry_price > 0.0 {
                    ((exit_price_f64 - entry_price) / entry_price) * 100.0
                } else {
                    0.0
                };
                (Some(exit_date), Some(exit_price_f64), Some(pnl_pct))
            } else {
                (None, None, None)
            };

            let was_traded = passed_score && pnl_pct.is_some();
            let rejection_reasons = if passed_score {
                Vec::new()
            } else {
                build_rejection_reasons(min_score, score.total, &score.explanation)
            };
            let strategy_checkpoints = build_strategy_checkpoints(&signal, &score.explanation, candle_signal.as_ref());

            reviews_for_date.push(BacktestCandidateReview {
                code: candidate.code.clone(),
                name: candidate.name.clone(),
                theme_name: Some(candidate.theme_name.clone()),
                status: if was_traded {
                    "pass".to_string()
                } else if passed_score {
                    "candidate".to_string()
                } else {
                    "fail".to_string()
                },
                passed_score,
                was_traded,
                score: score.total,
                score_theme: score.theme,
                score_news: score.news,
                score_technical: score.technical,
                score_risk: score.risk,
                change_rate: candidate.change_rate,
                news_count,
                ma5_support: score.ma5_support,
                reasoning: signal.reasoning.clone(),
                structured_reasoning,
                explanation: score.explanation.clone(),
                strategy_checkpoints,
                rejection_reasons,
                exit_date: exit_date.clone(),
                exit_price,
                pnl_pct,
                theme_rank: candidate.theme_rank,
                theme_stock_count: candidate.theme_stock_count,
                theme_positive_count: candidate.theme_positive_count,
                history,
                related_news: news,
                candle_signal,
            });

            if score.total < min_score {
                continue;
            }

            if let (Some(exit_date), Some(exit_price), Some(pnl_pct)) = (exit_date, exit_price, pnl_pct) {
                let entry_price = candidate
                    .current_price
                    .or(judal_db::get_price_on_date(&state.judal_db_path, &candidate.code, date)?)
                    .unwrap_or(100_000) as f64;

                trades.push(BacktestTrade {
                    id: 0,
                    run_id: 0,
                    code: candidate.code.clone(),
                    name: candidate.name.clone(),
                    theme_name: Some(candidate.theme_name.clone()),
                    entry_date: date.clone(),
                    exit_date,
                    entry_price,
                    exit_price,
                    pnl_pct,
                    score: Some(score.total),
                    score_theme: Some(score.theme),
                    score_news: Some(score.news),
                    score_technical: Some(score.technical),
                    news_count: Some(news_count),
                    ma5_support: Some(score.ma5_support),
                    explanation_summary: summarize_explanation(&score.explanation),
                    explanation: Some(score.explanation.clone()),
                });
            }
        }

        daily_candidate_reviews.push((date.clone(), reviews_for_date));

        on_progress((index as i64) + 1, total_steps, Some(date));
    }

    let total_trades = trades.len() as i64;
    let win_count = trades.iter().filter(|trade| trade.pnl_pct > 0.0).count() as i64;
    let loss_count = total_trades - win_count;
    let avg_return = if total_trades > 0 {
        trades.iter().map(|trade| trade.pnl_pct).sum::<f64>() / total_trades as f64
    } else {
        0.0
    };

    let mut equity = 1.0;
    let mut peak = 1.0;
    let mut max_drawdown = 0.0;
    for trade in &trades {
        equity *= 1.0 + (trade.pnl_pct / 100.0);
        if equity > peak {
            peak = equity;
        }
        let drawdown = if peak > 0.0 { ((peak - equity) / peak) * 100.0 } else { 0.0 };
        if drawdown > max_drawdown {
            max_drawdown = drawdown;
        }
    }
    let total_return = (equity - 1.0) * 100.0;
    let win_rate = if total_trades > 0 {
        (win_count as f64 / total_trades as f64) * 100.0
    } else {
        0.0
    };

    let run = BacktestRun {
        id: 0,
        run_at: run_timestamp.clone(),
        date_from: date_from.to_string(),
        date_to: date_to.to_string(),
        min_score,
        total_trades,
        win_count,
        loss_count,
        win_rate,
        avg_return,
        total_return,
        max_drawdown,
        status: "done".to_string(),
        created_at: run_timestamp,
    };

    let run_id = app_db::insert_backtest_run(&state.app_db_path, &run)?;
    for trade in &mut trades {
        trade.run_id = run_id;
        app_db::insert_backtest_trade(&state.app_db_path, trade)?;
    }
    for (entry_date, reviews) in &daily_candidate_reviews {
        app_db::replace_backtest_candidates(&state.app_db_path, run_id, entry_date, reviews)?;
    }

    on_progress(total_steps, total_steps, None);

    Ok(RunBacktestResponse {
        run: BacktestRun { id: run_id, ..run },
        trades,
    })
}

pub async fn run(state: &AppState, date_from: &str, date_to: &str, min_score: f64) -> AppResult<RunBacktestResponse> {
    run_with_progress(state, date_from, date_to, min_score, |_, _, _| {}).await
}
