# Rig AI Agent Patterns

## Overview

Rig is a Rust framework for building AI agents with tool-calling capabilities. The overnight module uses Rig to create an AI agent that analyzes stocks for overnight entry signals.

## Tool Definition

### Basic Tool Structure

```rust
use rig::tool::Tool;
use serde::{Deserialize, Serialize};

// 1. Define input schema
#[derive(Deserialize, schemars::JsonSchema)]
pub struct GetTrendingThemesInput {
    /// Maximum number of themes to return
    pub limit: Option<u32>,
}

// 2. Define output schema
#[derive(Serialize)]
pub struct Theme {
    pub theme_idx: u32,
    pub name: String,
    pub stock_count: u32,
    pub change_rate: f64,
}

// 3. Define error type
#[derive(Debug, thiserror::Error)]
#[error("Failed to fetch trending themes: {0}")]
pub struct GetTrendingThemesError(String);

// 4. Implement the tool
pub struct GetTrendingThemesTool {
    client: reqwest::Client,
    backend_url: String,
}

impl Tool for GetTrendingThemesTool {
    const NAME: &'static str = "get_trending_themes";
    
    type Input = GetTrendingThemesInput;
    type Output = Vec<Theme>;
    type Error = GetTrendingThemesError;
    
    async fn definition(&self, _prompt: String) -> rig::tool::ToolDefinition {
        rig::tool::ToolDefinition {
            name: Self::NAME.to_string(),
            description: "실시간 상승 테마 목록을 조회합니다. 테마명, 종목 수, 등락률을 반환합니다.".to_string(),
            parameters: serde_json::json!({
                "type": "object",
                "properties": {
                    "limit": {
                        "type": "integer",
                        "description": "반환할 최대 테마 수 (기본값: 10)"
                    }
                }
            }),
        }
    }
    
    async fn call(&self, input: Self::Input) -> Result<Self::Output, Self::Error> {
        let limit = input.limit.unwrap_or(10);
        let url = format!(
            "{}/judal/realtime/themes/rising?limit={}",
            self.backend_url, limit
        );
        
        let themes: Vec<Theme> = self.client
            .get(&url)
            .send()
            .await
            .map_err(|e| GetTrendingThemesError(e.to_string()))?
            .json()
            .await
            .map_err(|e| GetTrendingThemesError(e.to_string()))?;
        
        Ok(themes)
    }
}
```

## Overnight Agent Tools

### 1. Get Trending Themes

```rust
pub struct GetTrendingThemesTool { /* ... */ }
// GET /judal/realtime/themes/rising
```

### 2. Get Theme Stocks

```rust
#[derive(Deserialize, schemars::JsonSchema)]
pub struct GetThemeStocksInput {
    /// Theme index to query
    pub theme_idx: u32,
}

pub struct GetThemeStocksTool { /* ... */ }
// GET /judal/themes/{theme_idx}/stocks
```

### 3. Search News

```rust
#[derive(Deserialize, schemars::JsonSchema)]
pub struct SearchNewsInput {
    /// Search query (stock name or keyword)
    pub query: String,
    /// Maximum number of articles
    pub limit: Option<u32>,
}

pub struct SearchNewsTool {
    client: reqwest::Client,
    meili_url: String,
    meili_key: String,
}
// POST http://meilisearch:7700/indexes/articles/search
```

### 4. Get Stock History

```rust
#[derive(Deserialize, schemars::JsonSchema)]
pub struct GetStockHistoryInput {
    /// Stock code (e.g., "005930")
    pub code: String,
}

pub struct GetStockHistoryTool { /* ... */ }
// GET /judal/stocks/{code}/history
```

### 5. Score Overnight Signal

```rust
#[derive(Deserialize, schemars::JsonSchema)]
pub struct ScoreOvernightInput {
    pub code: String,
    pub name: String,
    pub theme_name: String,
    pub theme_rank: u32,
    pub news_count: u32,
    pub has_exclusive_news: bool,
    pub current_price: f64,
    pub ma5: f64,
    pub change_rate: f64,
    pub prev_change_rate: f64,
}

#[derive(Serialize)]
pub struct OvernightScore {
    pub total: f64,
    pub theme_score: f64,
    pub news_score: f64,
    pub technical_score: f64,
    pub risk_score: f64,
    pub reasoning: String,
}

pub struct ScoreOvernightTool;
// Pure computation, no external calls
```

## Agent Definition

### Creating the Agent

```rust
use rig::providers::openai;
use rig::agent::Agent;

pub async fn create_overnight_agent(config: &Config) -> anyhow::Result<Agent> {
    let client = openai::Client::new(&config.openai_api_key);
    
    let tools = vec![
        GetTrendingThemesTool::new(&config.backend_url),
        GetThemeStocksTool::new(&config.backend_url),
        SearchNewsTool::new(&config.meili_url, &config.meili_key),
        GetStockHistoryTool::new(&config.backend_url),
        ScoreOvernightTool::new(),
    ];
    
    let agent = client
        .agent("gpt-4o")
        .preamble(SYSTEM_PROMPT)
        .tools(tools)
        .build();
    
    Ok(agent)
}
```

### System Prompt

```rust
const SYSTEM_PROMPT: &str = r#"
당신은 한국 주식 시장 overnight 전략 전문가입니다.
장 마감(15:00–15:30) 직전 진입하여 다음날 시초가에 청산하는
하룻밤 보유 전략으로 수익을 내는 것이 목표입니다.

## 점수 기준 (총 10점)

### 테마 강도 (2점)
- 2.0점: 실시간 상승 테마 1위, 5개 이상 종목 동반 상승
- 1.5점: 상승 테마 3위 이내
- 1.0점: 테마 참여 종목
- 0.5점: 간접 연관

### 뉴스 품질 (3점)
- +1.5점: 오늘 단독 보도 / 정부 정책 연계 뉴스
- +1.0점: 최근 3일 이내 관련 기사 3건 이상
- +0.5점: 기사 1-2건
- -1.0점: 부정적 뉴스 (실적 쇼크, 소송, 사고)

### 기술적 분석 (3점)
- +1.0점: 당일 종가 > 5일 이동평균 (5일선 지지)
- +1.0점: 윗꼬리 없는 양봉 (종가 ≈ 고가)
- +1.0점: 거래대금 충분 (change_rate > 2% 또는 상승 지속)
- -0.5점: 장대 윗꼬리 (고가 대비 종가 3% 이상 하락)

### 리스크 (2점)
- 2.0점: 전일 등락률 -2% ~ +5% (과열 아님)
- 1.0점: 전일 +5% ~ +10% (주의)
- 0.0점: 전일 +10% 이상 또는 연속 3일 급등 (고위험)

## 출력 형식
반드시 JSON으로만 응답하세요. 설명은 reasoning 필드에 150자 이내로 작성.
"#;
```

## HTTP Endpoint

### Analyze Endpoint

```rust
use axum::{extract::State, Json};

#[derive(Deserialize)]
pub struct AnalyzeRequest {
    pub date: String,
    pub candidates: Vec<Candidate>,
}

#[derive(Deserialize)]
pub struct Candidate {
    pub code: String,
    pub name: String,
    pub theme_idx: u32,
    pub theme_name: String,
    pub current_price: f64,
    pub change_rate: f64,
    pub history_prices: Vec<f64>,
}

#[derive(Serialize)]
pub struct AnalyzeResponse {
    pub signals: Vec<SignalScore>,
}

pub async fn analyze(
    State(state): State<AgentState>,
    Json(req): Json<AnalyzeRequest>,
) -> Result<Json<AnalyzeResponse>, AppError> {
    let agent = &state.agent;
    
    let prompt = format!(
        "다음 종목들을 overnight 전략 기준으로 분석하고 점수를 매겨주세요:\n{}",
        serde_json::to_string_pretty(&req.candidates)?
    );
    
    let response = agent.chat(&prompt).await?;
    let signals: Vec<SignalScore> = serde_json::from_str(&response)?;
    
    Ok(Json(AnalyzeResponse { signals }))
}
```

## Best Practices

### 1. Tool Descriptions in Korean

For Korean stock analysis, write tool descriptions in Korean for better LLM understanding:

```rust
description: "종목 관련 뉴스를 검색합니다. 종목명이나 키워드로 검색할 수 있습니다.".to_string(),
```

### 2. Structured Output

Force JSON output with clear schema:

```rust
let prompt = format!(
    r#"
다음 종목을 분석하세요: {}

반드시 다음 JSON 형식으로만 응답하세요:
{{
  "code": "종목코드",
  "score": 0.0,
  "score_theme": 0.0,
  "score_news": 0.0,
  "score_technical": 0.0,
  "score_risk": 0.0,
  "reasoning": "150자 이내 요약"
}}
"#,
    stock_name
);
```

### 3. Error Recovery

Handle tool failures gracefully:

```rust
impl Tool for SearchNewsTool {
    async fn call(&self, input: Self::Input) -> Result<Self::Output, Self::Error> {
        match self.search_internal(&input).await {
            Ok(articles) => Ok(articles),
            Err(e) => {
                tracing::warn!("News search failed: {}, returning empty", e);
                Ok(vec![]) // Return empty instead of failing
            }
        }
    }
}
```

### 4. Rate Limiting

Respect external API limits:

```rust
use tokio::time::{sleep, Duration};

impl SearchNewsTool {
    async fn call(&self, input: Self::Input) -> Result<Self::Output, Self::Error> {
        // Simple rate limiting
        sleep(Duration::from_millis(100)).await;
        // ... actual call
    }
}
```
