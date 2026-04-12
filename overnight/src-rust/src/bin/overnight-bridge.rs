use std::env;
use std::io::{self, Write};

use overnight_lib::agent::reasoning::build_reasoning;
use overnight_lib::backtest::engine;
use overnight_lib::database::{app_db, candle_db, judal_db};
use overnight_lib::model::signal::{Signal, StrategyCheckpoint};
use overnight_lib::model::system::AppPaths;
use overnight_lib::service::app_state::AppState;
use overnight_lib::service::error::AppError;
use overnight_lib::service::scoring::calculate_score;
use serde_json::json;

fn print_json(value: serde_json::Value) {
    println!("{}", value);
}

fn flush_stdout() {
    let _ = io::stdout().flush();
}

fn first_non_empty(values: &[String], fallback: impl Into<String>) -> String {
    values
        .iter()
        .find(|value| !value.trim().is_empty())
        .cloned()
        .unwrap_or_else(|| fallback.into())
}

fn build_strategy_checkpoints(
    signal: &Signal,
    explanation: &overnight_lib::model::signal::ScoreExplanation,
    candle_signal: Option<&overnight_lib::model::signal::CandleSignal>,
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
            passed: !structured_reasoning.reasoning.trim().is_empty()
                || !signal.reasoning.trim().is_empty(),
            note: if !structured_reasoning.reasoning.trim().is_empty() {
                structured_reasoning.reasoning.clone()
            } else {
                signal.reasoning.clone()
            },
        },
    ]
}

#[tokio::main]
async fn main() {
    if let Err(error) = run().await {
        eprintln!("{error}");
        std::process::exit(1);
    }
}

async fn run() -> Result<(), String> {
    let args = env::args().skip(1).collect::<Vec<_>>();
    let Some(command) = args.first().cloned() else {
        return Err("missing command".to_string());
    };

    let state = AppState::new();
    state.ensure_data_dir().map_err(|error| error.to_string())?;
    app_db::initialize(&state.app_db_path).map_err(|error| error.to_string())?;
    state.validate_paths().map_err(|error| error.to_string())?;

    match command.as_str() {
        "health-check" => {
            print_json(json!({
                "service": "overnight-web",
                "status": "ok",
                "version": env!("CARGO_PKG_VERSION")
            }));
        }
        "app-paths" => {
            let app_paths = AppPaths {
                app_data_dir: state
                    .app_db_path
                    .parent()
                    .map(|path| path.display().to_string())
                    .unwrap_or_default(),
                overnight_db_path: state.app_db_path.display().to_string(),
                judal_db_path: state.judal_db_path.display().to_string(),
                candles_dir: Some(state.candles_dir.display().to_string()),
            };
            print_json(serde_json::to_value(app_paths).map_err(|error| error.to_string())?);
        }
        "available-dates" => {
            let dates = judal_db::get_history_dates(&state.judal_db_path, 365).map_err(|error| error.to_string())?;
            print_json(json!({ "dates": dates }));
        }
        "get-signals" => {
            let date = args.get(1).map(|value| value.as_str());
            let signals = app_db::list_signals(&state.app_db_path, date).map_err(|error| error.to_string())?;
            print_json(serde_json::to_value(signals).map_err(|error| error.to_string())?);
        }
        "generate-signals" => {
            let date = args.get(1).ok_or_else(|| "missing date".to_string())?;
            let min_score = args
                .get(2)
                .map(|value| value.parse::<f64>().map_err(|error| error.to_string()))
                .transpose()?
                .unwrap_or(8.0);

            let candidates = judal_db::get_candidates_for_date(&state.judal_db_path, date, None)
                .map_err(|error| error.to_string())?;
            app_db::delete_signals_for_date(&state.app_db_path, date).map_err(|error| error.to_string())?;

            for candidate in candidates {
                let recent_prices = judal_db::get_recent_prices(&state.judal_db_path, &candidate.code, date, 5)
                    .map_err(|error| error.to_string())?;
                let news = judal_db::search_related_news(
                    &state.http_client,
                    &state.meili_url,
                    &state.meili_api_key,
                    &candidate.name,
                    5,
                    Some(date),
                )
                .await
                .unwrap_or_default();
                let candle_signal = candle_db::get_latest_candle_signal(&state.candles_dir, &candidate.code, "KR")
                    .ok()
                    .flatten();

                let score = calculate_score(&candidate, &recent_prices, &news, candle_signal.as_ref());
                let news_titles = news
                    .iter()
                    .take(3)
                    .map(|article| article.title.clone())
                    .collect::<Vec<_>>();
                let reasoning = build_reasoning(&state.rig, &candidate, &score, &news_titles).await;
                let status = if score.total >= min_score { "pass" } else { "fail" };

                app_db::upsert_signal(
                    &state.app_db_path,
                    date,
                    &candidate,
                    &score,
                    &reasoning.reasoning,
                    &reasoning,
                    &score.explanation,
                    &news_titles,
                    news.len(),
                    status,
                )
                .map_err(|error| error.to_string())?;
            }

            let signals = app_db::list_signals(&state.app_db_path, Some(date)).map_err(|error| error.to_string())?;
            print_json(json!({
                "date": date,
                "generatedCount": signals.len(),
                "signals": signals,
            }));
        }
        "get-signal-detail" => {
            let code = args.get(1).ok_or_else(|| "missing code".to_string())?;
            let date = args.get(2).ok_or_else(|| "missing date".to_string())?;

            let signals = app_db::list_signals(&state.app_db_path, Some(date)).map_err(|error| error.to_string())?;
            let signal = signals
                .into_iter()
                .find(|signal| signal.code == *code)
                .ok_or_else(|| AppError::NotFound(format!("signal not found: {date} {code}")))
                .map_err(|error| error.to_string())?;

            let history = judal_db::get_stock_history(&state.judal_db_path, &signal.code, 10)
                .map_err(|error| error.to_string())?;
            let related_news = judal_db::search_related_news(
                &state.http_client,
                &state.meili_url,
                &state.meili_api_key,
                &signal.name,
                5,
                Some(date),
            )
            .await
            .unwrap_or_default();
            let recent_prices = judal_db::get_recent_prices(&state.judal_db_path, &signal.code, date, 5)
                .map_err(|error| error.to_string())?;
            let candle_signal = candle_db::get_latest_candle_signal(&state.candles_dir, &signal.code, "KR")
                .ok()
                .flatten();
            let candidate = judal_db::get_candidate_for_date(&state.judal_db_path, date, &signal.code)
                .map_err(|error| error.to_string())?
                .unwrap_or(overnight_lib::model::signal::SignalCandidate {
                    code: signal.code.clone(),
                    name: signal.name.clone(),
                    theme_idx: signal.theme_idx.unwrap_or_default(),
                    theme_name: signal.theme_name.clone().unwrap_or_default(),
                    current_price: signal.entry_price.map(|value| value as i64),
                    change_rate: signal.change_rate,
                    market_cap: history.first().and_then(|item| item.market_cap),
                    volume_index: None,
                    theme_stock_count: None,
                    theme_rank: None,
                    theme_positive_count: None,
                });
            let rescored = calculate_score(&candidate, &recent_prices, &related_news, candle_signal.as_ref());
            let strategy_checkpoints =
                build_strategy_checkpoints(&signal, &rescored.explanation, candle_signal.as_ref());

            print_json(json!({
                "signal": signal,
                "history": history,
                "relatedNews": related_news,
                "explanation": rescored.explanation,
                "candleSignal": candle_signal,
                "strategyCheckpoints": strategy_checkpoints,
            }));
        }
        "backtest-stats" => {
            let stats = app_db::get_backtest_stats(&state.app_db_path).map_err(|error| error.to_string())?;
            print_json(serde_json::to_value(stats).map_err(|error| error.to_string())?);
        }
        "list-backtest-runs" => {
            let runs = app_db::list_backtest_runs(&state.app_db_path).map_err(|error| error.to_string())?;
            print_json(serde_json::to_value(runs).map_err(|error| error.to_string())?);
        }
        "get-backtest-detail" => {
            let run_id = args.get(1).ok_or_else(|| "missing run_id".to_string())?;
            let run_id = run_id.parse::<i64>().map_err(|error| error.to_string())?;
            let detail = engine::get_backtest_detail(&state, run_id).map_err(|error| error.to_string())?;
            print_json(serde_json::to_value(detail).map_err(|error| error.to_string())?);
        }
        "get-backtest-day-review" => {
            let run_id = args.get(1).ok_or_else(|| "missing run_id".to_string())?;
            let date = args.get(2).ok_or_else(|| "missing date".to_string())?;
            let run_id = run_id.parse::<i64>().map_err(|error| error.to_string())?;
            let review = if let Some(review) = app_db::get_backtest_day_review(&state.app_db_path, run_id, date)
                .map_err(|error| error.to_string())?
            {
                review
            } else {
                engine::build_day_review(&state, run_id, date)
                    .await
                    .map_err(|error| error.to_string())?
            };
            print_json(serde_json::to_value(review).map_err(|error| error.to_string())?);
        }
        "run-backtest" => {
            let date_from = args.get(1).ok_or_else(|| "missing date_from".to_string())?;
            let date_to = args.get(2).ok_or_else(|| "missing date_to".to_string())?;
            let min_score = args
                .get(3)
                .ok_or_else(|| "missing min_score".to_string())?
                .parse::<f64>()
                .map_err(|error| error.to_string())?;

            let response = engine::run_with_progress(&state, date_from, date_to, min_score, |completed, total, current_date| {
                print_json(json!({
                    "type": "progress",
                    "completedSteps": completed,
                    "totalSteps": total,
                    "currentDate": current_date,
                }));
                flush_stdout();
            })
            .await
            .map_err(|error| error.to_string())?;

            print_json(json!({
                "type": "result",
                "payload": response,
            }));
        }
        _ => return Err(format!("unsupported command: {command}")),
    }

    Ok(())
}
