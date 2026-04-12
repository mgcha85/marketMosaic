use crate::model::signal::{CandleSignal, NewsArticle, ScoreExplanation, SignalCandidate};

#[derive(Debug, Clone)]
pub struct ScoreBreakdown {
    pub total: f64,
    pub theme: f64,
    pub news: f64,
    pub technical: f64,
    pub risk: f64,
    pub ma5: Option<f64>,
    pub ma5_support: bool,
    pub explanation: ScoreExplanation,
}

pub fn calculate_score(
    candidate: &SignalCandidate,
    recent_prices: &[i64],
    news: &[NewsArticle],
    candle_signal: Option<&CandleSignal>,
) -> ScoreBreakdown {
    let (theme, theme_reasons) = score_theme(candidate);
    let (technical, technical_reasons, ma5, ma5_support) = score_technical(
        candidate.current_price,
        recent_prices,
        candidate.volume_index,
        candle_signal,
    );
    let (news, news_reasons) = score_news(news);
    let (risk, risk_reasons) = score_risk(candidate.change_rate.unwrap_or_default());
    let total = (theme + technical + news + risk).min(10.0);

    ScoreBreakdown {
        total,
        theme,
        news,
        technical,
        risk,
        ma5,
        ma5_support,
        explanation: ScoreExplanation {
            theme_reasons,
            news_reasons,
            technical_reasons,
            risk_reasons,
        },
    }
}

fn score_theme(candidate: &SignalCandidate) -> (f64, Vec<String>) {
    let mut score: f64 = match candidate.change_rate.unwrap_or_default() {
        value if value >= 7.0 => 1.2,
        value if value >= 4.0 => 0.9,
        value if value > 0.0 => 0.6,
        _ => 0.2,
    };
    let mut reasons = Vec::new();

    match candidate.theme_rank.unwrap_or(99) {
        1 => {
            score += 0.6;
            reasons.push("전체 후보 1위 강도".to_string());
        }
        2 | 3 => {
            score += 0.4;
            reasons.push("전체 후보 상위 3위 내".to_string());
        }
        4..=6 => {
            score += 0.2;
            reasons.push("전체 후보 상위권".to_string());
        }
        _ => {}
    }

    let stock_count = candidate.theme_stock_count.unwrap_or_default();
    let positive_count = candidate.theme_positive_count.unwrap_or_default();
    if stock_count >= 5 {
        score += 0.1;
    }
    if stock_count > 0 {
        let breadth = positive_count as f64 / stock_count as f64;
        if breadth >= 0.8 {
            score += 0.3;
            reasons.push(format!("테마 확산 강함 ({positive_count}/{stock_count})"));
        } else if breadth >= 0.6 {
            score += 0.2;
            reasons.push(format!("테마 확산 양호 ({positive_count}/{stock_count})"));
        } else if breadth >= 0.4 {
            score += 0.1;
            reasons.push(format!(
                "테마 참여 종목 존재 ({positive_count}/{stock_count})"
            ));
        }
    }

    if let Some(change_rate) = candidate.change_rate {
        reasons.push(format!("등락률 {change_rate:.2}%"));
    }

    (score.min(2.0), reasons)
}

fn score_news(news: &[NewsArticle]) -> (f64, Vec<String>) {
    let deduped = dedupe_news(news);
    let count = deduped.len();
    let mut score = match count {
        n if n >= 5 => 1.5,
        3..=4 => 1.1,
        1..=2 => 0.6,
        _ => 0.0,
    };
    let mut reasons = vec![format!("중복 제거 후 기사 {count}건")];

    let freshest_hours = deduped.iter().filter_map(|item| item.age_hours).min();
    if let Some(hours) = freshest_hours {
        if hours <= 6 {
            score += 0.7;
            reasons.push("6시간 이내 기사 포함".to_string());
        } else if hours <= 24 {
            score += 0.4;
            reasons.push("24시간 이내 기사 포함".to_string());
        } else if hours <= 72 {
            score += 0.2;
            reasons.push("3일 이내 기사 포함".to_string());
        }
    }

    let mut positive_hits = 0;
    let mut negative_hits = 0;
    let mut trusted_publishers = 0;
    for article in &deduped {
        let text = article.title.to_lowercase();
        if contains_any(
            &text,
            &[
                "정책", "정부", "수주", "계약", "협약", "공급", "단독", "승인", "투자",
            ],
        ) {
            positive_hits += 1;
        }
        if contains_any(
            &text,
            &[
                "소송",
                "적자",
                "급락",
                "하향",
                "사고",
                "횡령",
                "유상증자",
                "실패",
            ],
        ) {
            negative_hits += 1;
        }
        if article
            .publisher
            .as_deref()
            .map(is_trusted_publisher)
            .unwrap_or(false)
        {
            trusted_publishers += 1;
        }
    }

    score += (positive_hits as f64 * 0.25).min(1.0);
    score -= (negative_hits as f64 * 0.5).min(1.0);
    score += (trusted_publishers as f64 * 0.1).min(0.4);

    if positive_hits > 0 {
        reasons.push(format!("정책/계약/단독 계열 키워드 {positive_hits}건"));
    }
    if trusted_publishers > 0 {
        reasons.push(format!("신뢰 발행사 기사 {trusted_publishers}건"));
    }
    if negative_hits > 0 {
        reasons.push(format!("악재 키워드 {negative_hits}건 감점"));
    }

    (score.clamp(0.0, 3.0), reasons)
}

fn dedupe_news(news: &[NewsArticle]) -> Vec<NewsArticle> {
    let mut seen = std::collections::HashSet::new();
    news.iter()
        .filter(|article| {
            let normalized = article
                .title
                .to_lowercase()
                .replace(' ', "")
                .replace('[', "")
                .replace(']', "")
                .replace('"', "");
            seen.insert(normalized)
        })
        .cloned()
        .collect()
}

fn is_trusted_publisher(publisher: &str) -> bool {
    let normalized = publisher.to_lowercase();
    [
        "연합뉴스",
        "한국경제",
        "매일경제",
        "머니투데이",
        "서울경제",
        "이데일리",
        "뉴스1",
    ]
    .iter()
    .any(|target| normalized.contains(&target.to_lowercase()))
}

fn contains_any(text: &str, keywords: &[&str]) -> bool {
    keywords.iter().any(|keyword| text.contains(keyword))
}

fn score_technical(
    current_price: Option<i64>,
    recent_prices: &[i64],
    volume_index: Option<f64>,
    candle_signal: Option<&CandleSignal>,
) -> (f64, Vec<String>, Option<f64>, bool) {
    if recent_prices.is_empty() {
        let volume_bonus = if volume_index.unwrap_or_default() >= 1.2 {
            0.8
        } else {
            0.3
        };
        let candle_bonus = candle_signal.map(score_candle_signal).unwrap_or_default();
        return (
            (volume_bonus + candle_bonus).min(3.0),
            vec!["히스토리 부족으로 거래강도 중심 평가".to_string()],
            None,
            false,
        );
    }

    let ma5 = recent_prices.iter().map(|price| *price as f64).sum::<f64>() / recent_prices.len() as f64;
    let current = current_price
        .or_else(|| recent_prices.first().copied())
        .unwrap_or_default() as f64;
    let ma5_support = current > 0.0 && current >= ma5;
    let mut reasons = Vec::new();
    let mut score: f64 = if ma5_support { 1.5 } else { 0.4 };
    if ma5_support {
        reasons.push("현재가가 MA5 위에서 지지".to_string());
    } else {
        reasons.push("현재가가 MA5 아래".to_string());
    }

    if volume_index.unwrap_or_default() >= 1.2 {
        score += 0.9;
        reasons.push("volume_index 강세".to_string());
    } else if volume_index.unwrap_or_default() >= 0.8 {
        score += 0.5;
        reasons.push("volume_index 보통 이상".to_string());
    } else {
        score += 0.2;
    }

    if current > 0.0 && ma5 > 0.0 {
        let premium = ((current - ma5) / ma5) * 100.0;
        if premium.abs() <= 3.0 {
            score += 0.4;
            reasons.push("현재가와 MA5 괴리 양호".to_string());
        }
    }

    if let Some(candle) = candle_signal {
        score += score_candle_signal(candle);
        if candle.bullish_close {
            reasons.push("최근 candle 양봉 마감".to_string());
        }
        if candle.upper_wick_ratio.unwrap_or(1.0) <= 0.25 {
            reasons.push("윗꼬리 짧은 candle".to_string());
        }
        if candle.above_vwap == Some(true) {
            reasons.push("종가가 VWAP 상회".to_string());
        }
        if candle.volume_spike == Some(true) {
            reasons.push("거래량 스파이크".to_string());
        }
        if candle.trade_count_spike == Some(true) {
            reasons.push("체결건수 스파이크".to_string());
        }
    }

    (score.min(3.0), reasons, Some(ma5), ma5_support)
}

fn score_candle_signal(candle_signal: &CandleSignal) -> f64 {
    let mut score = 0.0;
    if candle_signal.bullish_close {
        score += 0.3;
    }
    if candle_signal.upper_wick_ratio.unwrap_or(1.0) <= 0.25 {
        score += 0.2;
    }
    if candle_signal.positive_body_ratio.unwrap_or_default() >= 0.4 {
        score += 0.2;
    }
    if candle_signal.above_vwap == Some(true) {
        score += 0.15;
    }
    if candle_signal.volume_spike == Some(true) {
        score += 0.15;
    }
    if candle_signal.trade_count_spike == Some(true) {
        score += 0.1;
    }
    score
}

fn score_risk(change_rate: f64) -> (f64, Vec<String>) {
    if (-2.0..=5.0).contains(&change_rate) {
        (2.0, vec!["과열 아닌 안정 구간".to_string()])
    } else if (5.0..=10.0).contains(&change_rate) {
        (1.2, vec!["당일 상승폭 다소 큼".to_string()])
    } else if (10.0..=20.0).contains(&change_rate) {
        (0.9, vec!["강한 상승이지만 추세 구간으로 인정".to_string()])
    } else if change_rate > 20.0 {
        (0.6, vec!["급등 구간이라 추격 리스크 반영".to_string()])
    } else {
        (1.2, vec!["약세이나 과열은 아님".to_string()])
    }
}

#[cfg(test)]
mod tests {
    use super::calculate_score;
    use crate::model::signal::{NewsArticle, SignalCandidate};

    fn candidate(change_rate: f64, current_price: i64, volume_index: f64) -> SignalCandidate {
        SignalCandidate {
            code: "005930".to_string(),
            name: "삼성전자".to_string(),
            theme_idx: 214,
            theme_name: "반도체".to_string(),
            current_price: Some(current_price),
            change_rate: Some(change_rate),
            market_cap: Some(100_000_000),
            volume_index: Some(volume_index),
            theme_stock_count: Some(8),
            theme_rank: Some(1),
            theme_positive_count: Some(7),
        }
    }

    fn news(count: usize) -> Vec<NewsArticle> {
        (0..count)
            .map(|idx| NewsArticle {
                title: if idx == 0 {
                    "정부 정책 수혜 기대".to_string()
                } else {
                    format!("관련 뉴스 {idx}")
                },
                published_at: "2026-04-11T09:00:00+09:00".to_string(),
                publisher: None,
                url: None,
                age_hours: Some(2),
            })
            .collect()
    }

    #[test]
    fn strong_candidate_scores_high() {
        let score = calculate_score(
            &candidate(6.5, 10_500, 1.5),
            &[10_100, 10_150, 10_200, 10_250, 10_300],
            &news(5),
            None,
        );
        assert!(score.total >= 8.0);
        assert!(score.ma5_support);
    }

    #[test]
    fn overheated_candidate_loses_risk_points() {
        let score = calculate_score(
            &candidate(12.5, 10_500, 1.5),
            &[10_100, 10_150, 10_200, 10_250, 10_300],
            &news(5),
            None,
        );
        assert_eq!(score.risk, 0.0);
    }

    #[test]
    fn missing_history_keeps_technical_score_limited() {
        let score = calculate_score(&candidate(1.5, 10_000, 0.6), &[], &[], None);
        assert!(score.technical <= 1.0);
        assert!(!score.ma5_support);
    }
}
