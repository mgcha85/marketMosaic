use rig::providers::{ollama, openai};
use tokio::time::{timeout, Duration};

use crate::model::signal::{SignalCandidate, StructuredReasoning};
use crate::service::app_state::RigConfig;
use crate::service::scoring::ScoreBreakdown;

pub async fn build_reasoning(
    rig: &RigConfig,
    candidate: &SignalCandidate,
    score: &ScoreBreakdown,
    news_titles: &[String],
) -> StructuredReasoning {
    let fallback = build_reasoning_fast(candidate, score, news_titles);

    match llm_reasoning(rig, candidate, score, news_titles).await {
        Ok(reasoning) if !reasoning.reasoning.trim().is_empty() => reasoning,
        _ => fallback,
    }
}

pub fn build_reasoning_fast(
    candidate: &SignalCandidate,
    score: &ScoreBreakdown,
    news_titles: &[String],
) -> StructuredReasoning {
    let mut parts = vec![format!("{} 테마 연관", candidate.theme_name)];

    if score.ma5_support {
        parts.push("5일선 지지 확인".to_string());
    }

    if let Some(change_rate) = candidate.change_rate {
        if change_rate > 0.0 {
            parts.push(format!("등락률 +{change_rate:.2}%"));
        }
    }

    if !news_titles.is_empty() {
        parts.push(format!("관련 뉴스 {}건", news_titles.len()));
    }

    let mut key_news_points = score.explanation.news_reasons.iter().take(2).cloned().collect::<Vec<_>>();
    if key_news_points.is_empty() {
        key_news_points = news_titles.iter().take(2).cloned().collect();
    }

    let mut caution_points = score.explanation.risk_reasons.clone();
    if score.risk < 1.5 {
        caution_points.push("리스크 점수가 보수적으로 계산됨".to_string());
    }
    if candidate.change_rate.unwrap_or_default() > 8.0 {
        caution_points.push("단기 과열 가능성 점검 필요".to_string());
    }

    StructuredReasoning {
        reasoning: parts.join(". "),
        key_news_points,
        caution_points,
    }
}

async fn llm_reasoning(
    rig: &RigConfig,
    candidate: &SignalCandidate,
    score: &ScoreBreakdown,
    news_titles: &[String],
) -> Result<StructuredReasoning, String> {
    let prompt = format!(
        "당신은 한국 주식 overnight 전략 분석가입니다. 다음 데이터를 바탕으로 구조화된 JSON을 생성하세요.\n\n종목명: {}\n종목코드: {}\n테마: {}\n총점: {:.1}\n테마점수: {:.1}\n뉴스점수: {:.1}\n기술점수: {:.1}\n리스크점수: {:.1}\n5일선지지: {}\n테마근거: {}\n뉴스근거: {}\n기술근거: {}\n리스크근거: {}\n뉴스제목: {}\n등락률: {}",
        candidate.name,
        candidate.code,
        candidate.theme_name,
        score.total,
        score.theme,
        score.news,
        score.technical,
        score.risk,
        if score.ma5_support { "예" } else { "아니오" },
        score.explanation.theme_reasons.join(" | "),
        score.explanation.news_reasons.join(" | "),
        score.explanation.technical_reasons.join(" | "),
        score.explanation.risk_reasons.join(" | "),
        if news_titles.is_empty() {
            "없음".to_string()
        } else {
            news_titles.join(" | ")
        },
        candidate
            .change_rate
            .map(|value| format!("{value:.2}%"))
            .unwrap_or_else(|| "정보없음".to_string())
    );

    if rig.api_key.is_empty() {
        let client = ollama::Client::from_url(rig.base_url.as_str());
        let extractor = client
            .extractor::<StructuredReasoning>(rig.model_name.as_str())
            .preamble("당신은 한국 주식 overnight 전략 reasoning 생성기입니다.")
            .build();

        timeout(Duration::from_secs(12), extractor.extract(&prompt))
            .await
            .map_err(|_| "reasoning timed out".to_string())?
            .map_err(|error| error.to_string())
    } else {
        let normalized_base_url = normalize_openai_base_url(rig.base_url.as_str());
        let client = openai::Client::from_url(rig.api_key.as_str(), &normalized_base_url);
        let extractor = client
            .extractor::<StructuredReasoning>(rig.model_name.as_str())
            .preamble("당신은 한국 주식 overnight 전략 reasoning 생성기입니다.")
            .build();

        timeout(Duration::from_secs(12), extractor.extract(&prompt))
            .await
            .map_err(|_| "reasoning timed out".to_string())?
            .map_err(|error| error.to_string())
    }
}

fn normalize_openai_base_url(base_url: &str) -> String {
    let trimmed = base_url.trim_end_matches('/');
    if trimmed.ends_with("/v1") {
        trimmed.to_string()
    } else {
        format!("{trimmed}/v1")
    }
}
