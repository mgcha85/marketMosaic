# Overnight 전략 검증 모듈 기획서

> **작성일**: 2026-04-11  
> **목적**: Judal 테마주 + 뉴스 + 캔들 분석을 결합한 overnight 진입 전략을 AI 에이전트로 자동화하고 백테스트로 검증  
> **스택**: Rust (백엔드), Rust/Rig (AI Agent), Svelte (프론트엔드), SQLite (overnight.db)

---

## 1. 전략 정의 (strategies.md 기반)

### Overnight 전략 핵심 원칙

| 원칙 | 내용 |
|------|------|
| **진입 시간** | 오후 3:00–3:30 (세력의 의도가 확정되는 장 마감 직전) |
| **청산 시간** | 다음날 오전 9:00–9:30 (시초가 또는 갭 확인 후) |
| **기대 수익** | 하룻밤 테마 모멘텀에 의한 갭상승 |

### 진입 후보 선정 기준 (점수제, 10점 만점)

```
[테마 강도   2점] 실시간 상승 테마 참여 종목 여부, 테마 내 순위
[뉴스 품질   3점] 관련 기사 수, 보도 신선도, 정책/단독 뉴스 여부
[기술적 분석 3점] 5일선 지지, 당일 거래대금, 윗꼬리 없는 캔들 형태
[리스크 역산 2점] 전일 급등 없음(연속 급등 회피), 52주 고가 대비 위치
```

**진입 조건**: 총점 **8점 이상** + 당일 **거래대금 500억 이상** (데이터 빠진 경우 change_rate > 0 대체)

### 청산 조건

- **익절**: 다음날 시초가 갭상승 +2% 이상 → 시초가 매도
- **손절**: 다음날 시초가 갭하락 -3% 이하 → 시초가 손절
- **기본**: 9:15 강제 청산 (데이터 기준 다음 날 스냅샷 가격)

---

## 2. 시스템 아키텍처

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MarketMosaic                                 │
│                                                                     │
│  ┌────────────────┐      ┌──────────────────────────────────────┐  │
│  │  Svelte UI     │◄────►│  Rust Backend (dx-unified :8080)        │  │
│  │  :8090         │      │                                      │  │
│  │  - Overnight   │      │  internal/overnight/                  │  │
│  │    Dashboard   │      │  ├─ api/handlers.rs                  │  │
│  │  - Backtest    │      │  ├─ database/ (overnight.db)          │  │
│  │    Results     │      │  ├─ model/types.rs                   │  │
│  └────────────────┘      │  ├─ service/service.rs               │  │
│                          │  └─ backtest/engine.rs               │  │
│                          │                      │               │  │
│                          │         HTTP POST     │               │  │
│                          │         /analyze      ▼               │  │
│                          │      ┌───────────────────────────┐   │  │
│                          │      │  Rig Agent Service (Rust)  │   │  │
│                          │      │  overnight/agent/ :8085    │   │  │
│                          │      │                           │   │  │
│                          │      │  Tools:                   │   │  │
│                          │      │  - get_trending_themes     │   │  │
│                          │      │  - search_news             │   │  │
│                          │      │  - analyze_candles         │   │  │
│                          │      │  - score_overnight         │   │  │
│                          │      │                           │   │  │
│                          │      │  LLM: OpenAI/Claude       │   │  │
│                          └──────┴───────────────────────────┘   │  │
│                                                                     │
│  ┌──────────────┐  ┌────────────────┐  ┌────────────────────────┐  │
│  │  judal.db    │  │  Meilisearch   │  │  Candle (Parquet/DuckDB)│  │
│  │  SQLite      │  │  :7700         │  │  backend/data/candles/ │  │
│  │  테마/종목   │  │  136k 뉴스     │  │  OHLCV 분봉/일봉       │  │
│  └──────────────┘  └────────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. 디렉토리 구조

```
marketMosaic/
├── overnight/                          # 신규 submodule
│   └── agent/                          # Rig AI Agent (Rust)
│       ├── Cargo.toml
│       └── src/
│           ├── main.rs                 # Axum HTTP server (:8085)
│           ├── agent.rs                # Rig agent 정의 + 실행 루프
│           ├── tools.rs                # Agent tools (Judal, Meili, Candle)
│           └── types.rs                # 요청/응답 타입
│
backend/
└── internal/
    └── overnight/                      # 신규 Rsut 패키지
        ├── api/
        │   └── handlers.rs             # REST 엔드포인트
        ├── database/
        │   ├── db.rs                   # overnight.db 초기화
        │   └── schema.rs               # DDL
        ├── model/
        │   └── types.rs                # Signal, Trade, BacktestRun
        ├── service/
        │   └── service.rs              # 비즈니스 로직 + Rig 호출
        └── backtest/
            └── engine.rs               # 백테스트 엔진

frontend/src/lib/components/
└── OvernightDashboard.svelte           # 신규 Svelte 컴포넌트
```

---

## 4. 데이터베이스 스키마 (overnight.db)

```sql
-- =============================================
-- 진입 신호 (AI가 선정한 overnight 후보)
-- =============================================
CREATE TABLE IF NOT EXISTS signals (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    signal_date     DATE    NOT NULL,               -- 진입 날짜 (ex: 2026-04-11)
    code            TEXT    NOT NULL,               -- 종목코드
    name            TEXT    NOT NULL,               -- 종목명
    theme_idx       INTEGER,                        -- 테마 인덱스
    theme_name      TEXT,                           -- 테마명
    score           REAL    NOT NULL,               -- AI 종합점수 (0-10)
    score_theme     REAL,                           -- 세부: 테마 강도
    score_news      REAL,                           -- 세부: 뉴스 품질
    score_technical REAL,                           -- 세부: 기술적 분석
    score_risk      REAL,                           -- 세부: 리스크
    reasoning       TEXT,                           -- AI 추론 요약
    entry_price     REAL,                           -- 진입 예상가 (종가)
    change_rate     REAL,                           -- 당일 등락률
    ma5             REAL,                           -- 5일 이동평균
    ma5_support     INTEGER DEFAULT 0,              -- 5일선 지지 여부 (0/1)
    news_count      INTEGER DEFAULT 0,              -- 관련 뉴스 수
    news_titles     TEXT,                           -- 대표 뉴스 JSON (top 3)
    status          TEXT    DEFAULT 'pending',      -- pending|confirmed|cancelled
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(signal_date, code)
);

-- =============================================
-- 백테스트 실행 기록
-- =============================================
CREATE TABLE IF NOT EXISTS backtest_runs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    run_at          DATETIME NOT NULL,
    date_from       DATE,                           -- 백테스트 시작일
    date_to         DATE,                           -- 백테스트 종료일
    min_score       REAL    DEFAULT 8.0,            -- 최소 점수 임계값
    total_trades    INTEGER DEFAULT 0,
    win_count       INTEGER DEFAULT 0,
    loss_count      INTEGER DEFAULT 0,
    win_rate        REAL,                           -- 승률 (%)
    avg_return      REAL,                           -- 평균 수익률 (%)
    total_return    REAL,                           -- 누적 수익률 (%)
    max_drawdown    REAL,                           -- 최대 낙폭 (%)
    status          TEXT    DEFAULT 'running',      -- running|done|failed
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- 백테스트 개별 거래
-- =============================================
CREATE TABLE IF NOT EXISTS backtest_trades (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id          INTEGER NOT NULL REFERENCES backtest_runs(id),
    code            TEXT    NOT NULL,
    name            TEXT    NOT NULL,
    theme_name      TEXT,
    entry_date      DATE    NOT NULL,
    exit_date       DATE    NOT NULL,
    entry_price     REAL    NOT NULL,               -- 진입일 종가
    exit_price      REAL    NOT NULL,               -- 익일 종가 (스냅샷 기준)
    pnl_pct         REAL    NOT NULL,               -- 수익률 (%)
    score           REAL,
    score_theme     REAL,
    score_news      REAL,
    score_technical REAL,
    news_count      INTEGER,
    ma5_support     INTEGER
);

CREATE INDEX IF NOT EXISTS idx_signals_date  ON signals(signal_date);
CREATE INDEX IF NOT EXISTS idx_bt_trades_run ON backtest_trades(run_id);
```

---

## 5. API 설계

### 5-1. Rig Agent Service (`overnight/agent`, port 8085)

| Method | Path | 설명 |
|--------|------|------|
| `POST` | `/analyze` | 종목 리스트를 받아 overnight 점수 + 추론 반환 |
| `GET`  | `/health` | 헬스체크 |

**Request Body** (`POST /analyze`):
```json
{
  "date": "2026-04-11",
  "candidates": [
    {
      "code": "005930",
      "name": "삼성전자",
      "theme_idx": 214,
      "theme_name": "반도체",
      "current_price": 62000,
      "change_rate": 3.2,
      "history_prices": [60000, 61000, 61500, 61000, 62000]
    }
  ]
}
```

**Response**:
```json
{
  "signals": [
    {
      "code": "005930",
      "score": 8.5,
      "score_theme": 1.8,
      "score_news": 2.7,
      "score_technical": 2.5,
      "score_risk": 1.5,
      "reasoning": "반도체 테마 상위 종목. 3개 관련 뉴스(단독 보도 포함). 5일선 지지 확인. 전일 급등 없음.",
      "ma5": 61300.0,
      "ma5_support": true,
      "news_count": 3,
      "news_titles": ["삼성전자 HBM 공급 계약 확대", "...]
    }
  ]
}
```

### 5-2. Go Backend (`/overnight/*`)

| Method | Path | 설명 |
|--------|------|------|
| `GET`  | `/overnight/signals` | 날짜별 신호 목록 (`?date=2026-04-11`) |
| `POST` | `/overnight/signals/generate` | 신호 생성 (Rig 에이전트 호출) |
| `GET`  | `/overnight/backtest` | 백테스트 실행 기록 목록 |
| `POST` | `/overnight/backtest/run` | 백테스트 실행 |
| `GET`  | `/overnight/backtest/:id` | 백테스트 상세 (개별 거래 포함) |
| `GET`  | `/overnight/stats` | 전체 백테스트 통계 요약 |

---

## 6. Rig Agent 상세 설계

### 6-1. Tools 목록

```rust
// Tool 1: 실시간 상승 테마 조회
async fn get_trending_themes(limit: u32) -> Vec<Theme>
// → GET /judal/realtime/themes/rising

// Tool 2: 테마별 종목 목록 조회  
async fn get_theme_stocks(theme_idx: u32) -> Vec<Stock>
// → GET /judal/themes/{theme_idx}/stocks

// Tool 3: 종목 관련 뉴스 검색
async fn search_news(query: String, limit: u32) -> Vec<Article>
// → POST http://meilisearch:7700/indexes/articles/search

// Tool 4: 종목 히스토리 (5일 가격 추이)
async fn get_stock_history(code: String) -> Vec<StockSnapshot>
// → GET /judal/stocks/{code}/history

// Tool 5: 캔들 패턴 분석 (일봉)
async fn analyze_daily_candles(code: String) -> CandleAnalysis
// → GET /candle/daily/{code}
```

### 6-2. Agent System Prompt

```
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
```

---

## 7. 백테스트 엔진 로직

### 데이터 제약 및 해결책

| 항목 | 상황 | 대응 |
|------|------|------|
| 종가 데이터 | Judal stock_history 12개 날짜 스냅샷 | `current_price`를 종가 대용 사용 |
| 시초가 데이터 | 없음 | 익일 `current_price` 기준 수익률 계산 |
| 뉴스 시간 데이터 | published_at 있음 | 진입일 15:00 이전 기사만 카운트 |
| 캔들 데이터 | 2026-01 KR 일부만 존재 | 백테스트는 judal 스냅샷만으로 수행 |

### 엔진 흐름

```
for each date D in sorted(judal stock_history dates):
    next_date = 다음 사용 가능한 날짜

    if next_date is None:
        continue  ← 마지막 날짜는 exit 불가

    for each stock in D의 상위 테마 종목:
        # 1. 테마 점수 계산
        theme_rank = 해당 날짜 테마 내 change_rate 순위
        score_theme = rank_to_score(theme_rank)

        # 2. 뉴스 점수 계산  
        news = meilisearch.search(stock.name, date=D, hour_to=15)
        score_news = news_scoring(news)

        # 3. 기술적 점수 계산
        history = 직전 5개 스냅샷 가격
        ma5 = mean(history[-5:])
        score_technical = technical_scoring(stock, ma5)

        # 4. 리스크 점수
        score_risk = risk_scoring(stock.change_rate, prev_change_rate)

        total_score = score_theme + score_news + score_technical + score_risk

        if total_score >= min_score (default: 8.0):
            entry_price = stock.current_price (D날 종가)
            exit_price  = next_date 해당 종목 current_price
            pnl_pct     = (exit_price - entry_price) / entry_price * 100

            backtest_trades.insert(...)

    backtest_runs.update(stats...)  ← 승률, 평균수익률, 누적수익률
```

### 성과 지표

```
승률(Win Rate)     = win_count / total_trades × 100
평균수익률         = mean(pnl_pct for all trades)
누적수익률         = (1 + r₁)(1 + r₂)...(1 + rₙ) - 1  (균등 배분 가정)
최대낙폭(MDD)      = max drawdown of cumulative equity curve
```

---

## 8. 프론트엔드 UI 설계

### 진입점

기존 `App.svelte`의 탭에 `#overnight` 해시 추가

### OvernightDashboard.svelte 레이아웃

```
┌─────────────────────────────────────────────────────┐
│  🌙 Overnight 전략                  [신호 생성] 버튼 │
│  2026-04-11                                         │
├─────────────────────────────────────────────────────┤
│  📊 오늘의 진입 후보 (N개)                           │
│                                                     │
│  ┌────────────────────────────────────────────────┐ │
│  │ [점수 8.5] 삼성전자 (005930)                   │ │
│  │ 테마: 반도체  |  등락률: +3.2%  |  5일선: 지지 │ │
│  │                                                │ │
│  │ 🤖 AI 추론:                                    │ │
│  │ "반도체 테마 상위. HBM 관련 단독 보도. 5일선   │ │
│  │  지지 확인. 리스크 낮음."                      │ │
│  │                                                │ │
│  │ 📰 관련 뉴스 (3건):                            │ │
│  │  • 삼성전자 HBM 공급 계약 확대                 │ │
│  │  • SK하이닉스 대비 수급 유입                   │ │
│  │                                                │ │
│  │ [점수 세부] [캔들 보기]                        │ │
│  └────────────────────────────────────────────────┘ │
│  (카드 반복...)                                      │
├─────────────────────────────────────────────────────┤
│  📈 백테스트 결과                    [백테스트 실행] │
│                                                     │
│  ┌──────────┬──────────┬──────────┬──────────────┐  │
│  │ 총 거래  │ 승률     │ 평균수익 │ 누적수익률   │  │
│  │ 47건     │ 62.5%    │ +1.8%    │ +18.4%       │  │
│  └──────────┴──────────┴──────────┴──────────────┘  │
│                                                     │
│  [수익곡선 라인차트]                                 │
│                                                     │
│  [개별 거래 테이블]                                  │
│  날짜 | 종목 | 테마 | 점수 | 진입가 | 익일가 | 수익률│
└─────────────────────────────────────────────────────┘
```

### 세부 컴포넌트

| 컴포넌트 | 역할 |
|---------|------|
| `SignalCard.svelte` | 종목 점수 카드 (점수 게이지, 추론, 뉴스) |
| `ScoreBreakdown.svelte` | 4가지 세부 점수 레이더/바 차트 |
| `BacktestSummary.svelte` | 성과 지표 카드 (승률, 수익) |
| `EquityCurve.svelte` | 누적 수익 라인차트 (lightweight-charts) |
| `BacktestTradeTable.svelte` | 개별 거래 테이블 (정렬/필터 지원) |

---

## 9. Rig Agent Cargo.toml 의존성

```toml
[package]
name = "overnight-agent"
version = "0.1.0"
edition = "2021"

[dependencies]
rig-core = "0.9"          # AI agent framework
axum = "0.8"              # HTTP server
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
reqwest = { version = "0.12", features = ["json"] }
tracing = "0.1"
tracing-subscriber = "0.3"
anyhow = "1"
```

---

## 10. Docker Compose 통합

`docker-compose.yml`에 추가:

```yaml
  overnight-agent:
    build: ./overnight/agent
    ports:
      - "8085:8085"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - BACKEND_URL=http://dx-unified:8080
      - MEILI_HOST=http://meilisearch:7700
      - MEILI_API_KEY=${MEILI_API_KEY:-masterKey}
      - RUST_LOG=info
    depends_on:
      - dx-unified
      - meilisearch
    restart: unless-stopped
```

`backend/.env`에 추가:
```
OVERNIGHT_AGENT_URL=http://localhost:8085
OVERNIGHT_DB_PATH=./data/overnight.db
```

---

## 11. 구현 단계 (Phase)

### Phase 1: 백테스트 엔진 + DB (Rust)
- `internal/overnight/database/` - overnight.db 스키마 + 초기화
- `internal/overnight/backtest/engine.rs` - 백테스트 로직 (Judal + Meili 연동)
- `internal/overnight/api/handlers.rs` - REST 엔드포인트
- `cmd/server/main.rs` - overnight 라우터 등록
- **출력물**: `GET /overnight/backtest/run` 동작 확인

### Phase 2: Rig AI Agent (Rust)
- `overnight/agent/Cargo.toml` 및 의존성 설정
- `src/types.rs` - 요청/응답 타입
- `src/tools.rs` - 5개 Tool 구현 (Judal, Meili, Candle API 호출)
- `src/agent.rs` - Rig 에이전트 + 프롬프트
- `src/main.rs` - Axum HTTP 서버 (:8085)
- **출력물**: `POST /analyze` 동작 확인

### Phase 3: 신호 생성 서비스 (Rust)
- `internal/overnight/service/service.rs` - Rig 에이전트 HTTP 호출 + DB 저장
- `GET /overnight/signals/generate` 엔드포인트
- **출력물**: AI 기반 신호 생성 + overnight.db 저장 확인

### Phase 4: 프론트엔드 (Svelte)
- `OvernightDashboard.svelte` - 메인 대시보드
- `SignalCard.svelte`, `BacktestSummary.svelte` 등 하위 컴포넌트
- `App.svelte` Navbar + 탭 연결
- **출력물**: 브라우저에서 신호 카드 + 백테스트 결과 확인

### Phase 5: Docker 통합 + 최종 테스트
- `docker-compose.yml` overnight-agent 서비스 추가
- 전체 E2E 흐름 검증

---

## 12. 제약 사항 및 주의점

| 항목 | 내용 |
|------|------|
| **데이터 한계** | Judal 스냅샷 12개 날짜 → 최대 11개 overnight 거래 시뮬레이션 가능 |
| **가격 신뢰도** | 분봉 데이터 없어 종가→익일종가로 수익률 계산 (시초가 갭 미반영) |
| **뉴스 타이밍** | Meili 뉴스는 published_at으로 필터링하나 15:00 이전/이후 구분 한계 |
| **LLM 비용** | Rig 에이전트는 종목당 1 LLM 호출 → 후보 20개 기준 20 calls/day |
| **백테스트 신뢰도** | 12개 날짜 스냅샷이므로 통계적 유의성 낮음, 추후 데이터 누적 시 재검증 필요 |

---

## 13. 빠른 참조

```bash
# 신호 생성 (AI 분석)
curl -X POST http://localhost:8080/overnight/signals/generate \
  -H "Content-Type: application/json" \
  -d '{"date": "2026-04-11", "min_score": 8.0}'

# 신호 조회
curl "http://localhost:8080/overnight/signals?date=2026-04-11"

# 백테스트 실행
curl -X POST http://localhost:8080/overnight/backtest/run \
  -H "Content-Type: application/json" \
  -d '{"date_from": "2026-01-06", "date_to": "2026-04-11", "min_score": 8.0}'

# 백테스트 결과
curl http://localhost:8080/overnight/backtest/1

# Rig 에이전트 직접 호출 (개발/디버그)
curl -X POST http://localhost:8085/analyze \
  -H "Content-Type: application/json" \
  -d '{"date": "2026-04-11", "candidates": [...]}'
```
