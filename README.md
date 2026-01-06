# DX Unified

4개의 독립적인 Go 백엔드 프로젝트를 하나로 통합한 데이터 수집 및 API 서버입니다.

## 통합 서비스

| 서비스 | 설명 | API Prefix |
|--------|------|-----------|
| **DART** | 한국 금융감독원 공시 수집 | `/dart/*` |
| **Judal** | judal.co.kr 주식 테마/종목 크롤링 | `/judal/*` |
| **Candle** | KR/US 시장 캔들 데이터 수집 | `/candle/*` |
| **News** | 경제 뉴스 수집 (네이버, NewsAPI) | `/news/*` |

## 빠른 시작

### 1. 환경 설정

```bash
cp .env.example .env
# .env 파일을 열어 API 키들을 설정하세요
```

### 2. 빌드 및 실행

```bash
# 로컬 실행
go build -o dx-unified ./cmd/server
./dx-unified

# Docker Compose 실행
docker-compose up -d
```

### 3. API 테스트

```bash
# 헬스체크
curl http://localhost:8080/health

# DART 기업 목록
curl http://localhost:8080/dart/corps

# Judal 상승 테마 (실시간)
curl http://localhost:8080/judal/realtime/themes/rising

# 캔들 데이터
curl "http://localhost:8080/candle/stocks?market=KR&symbol=005930&timeframe=1d"

# 뉴스 검색
curl "http://localhost:8080/news/search?q=삼성전자"
```

---

## API 엔드포인트

### DART (`/dart/*`)

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/dart/corps` | 기업 목록 (page, limit) |
| GET | `/dart/filings` | 공시 목록 (corp_code, stock_code, date_from, date_to) |
| GET | `/dart/filings/:rcept_no` | 공시 상세 |

### Judal (`/judal/*`)

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/judal/themes` | 전체 테마 목록 |
| GET | `/judal/themes/:idx` | 특정 테마 조회 |
| GET | `/judal/themes/:idx/stocks` | 테마별 종목 |
| GET | `/judal/stocks` | 종목 목록 (sort, order, market, limit) |
| GET | `/judal/stocks/:code` | 종목 상세 |
| GET | `/judal/stocks/:code/history` | 종목 히스토리 |
| GET | `/judal/realtime/tabs` | 사용 가능한 탭 목록 |
| GET | `/judal/realtime/themes/:tab` | 테마 탭 실시간 크롤링 |
| GET | `/judal/realtime/stocks/:tab` | 종목 탭 실시간 크롤링 |
| POST | `/judal/crawl` | 크롤링 시작 |
| POST | `/judal/crawl/batch` | 일배치 크롤링 |
| GET | `/judal/status` | 크롤러 상태 |

**Realtime 탭 목록:**
- 테마: `all`, `rising`, `falling`, `expected`, `hot`, `neglected`
- 종목: `rising`, `falling`, `low_pbr`, `low_per`, `high_expected`, `fund_buy`, `foreign_buy` 등

### Candle (`/candle/*`) - DuckDB + Parquet/Hive

> 캔들 데이터는 Hive 파티션 구조 (`market={market}/year=YYYY/month=MM/*.parquet`)로 저장되며, DuckDB로 쿼리합니다.

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/candle/universe` | 유니버스 목록 (market, limit) |
| GET | `/candle/stocks` | 캔들 데이터 (market, symbol, timeframe, date_from, date_to) |
| GET | `/candle/stocks/:symbol` | 특정 종목 캔들 |
| GET | `/candle/dates` | 사용 가능한 날짜 목록 |
| GET | `/candle/runs` | 수집 실행 로그 |

### News (`/news/*`)

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/news/articles` | 뉴스 목록 (source, keyword, limit) |
| GET | `/news/articles/:id` | 뉴스 상세 |
| GET | `/news/search` | 뉴스 검색 (q) |
| GET | `/news/runs` | 배치 실행 로그 |

---

## 배치 스케줄

통합 스케줄러가 모든 배치 작업을 관리합니다:

| 작업 | 스케줄 | 설명 |
|------|--------|------|
| DART-FetchFilings | 매시간 | 최근 3일 공시 목록 수집 |
| DART-DownloadDocs | 5분마다 | 미다운로드 문서 다운로드 |
| DART-UpdateCorpCodes | 매주 | 기업코드 업데이트 |
| Judal-DailyCrawl | 16:00 KST | 전체 테마/종목 크롤링 |
| Candle-IngestKR | 20:00 KST | 한국 시장 캔들 수집 |
| Candle-IngestUS | 20:00 ET | 미국 시장 캔들 수집 |
| News-FetchNews | 15분마다 | 뉴스 수집 |

---

## 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `PORT` | 8080 | 서버 포트 |
| `DART_DB_PATH` | ./data/dart.db | DART SQLite 경로 |
| `JUDAL_DB_PATH` | ./data/judal.db | Judal SQLite 경로 |
| `CANDLE_DATA_DIR` | ./data/candles | Candle Parquet 데이터 디렉토리 (Hive 파티션) |
| `MEILI_HOST` | http://localhost:7700 | Meilisearch 호스트 |
| `MEILI_API_KEY` | masterKey | Meilisearch API 키 |
| `DART_API_KEY` | - | DART API 키 (금융감독원) |
| `KIWOOM_APP_KEY` | - | Kiwoom 앱 키 |
| `KIWOOM_APP_SECRET` | - | Kiwoom 앱 시크릿 |
| `ALPACA_API_KEY` | - | Alpaca API 키 |
| `ALPACA_API_SECRET` | - | Alpaca API 시크릿 |
| `FMP_API_KEY` | - | FMP API 키 |
| `NAVER_CLIENT_ID` | - | 네이버 클라이언트 ID |
| `NAVER_CLIENT_SECRET` | - | 네이버 클라이언트 시크릿 |
| `NEWSAPI_KEY` | - | NewsAPI 키 |

---

## 프로젝트 구조

```
dx-unified/
├── cmd/server/main.go           # 통합 진입점
├── internal/
│   ├── dart/                    # DART 모듈
│   │   ├── api/                 # API 핸들러
│   │   ├── database/            # DB 레이어
│   │   ├── models/              # 데이터 모델
│   │   └── scheduler/           # 배치 로직
│   ├── judal/                   # Judal 모듈
│   │   ├── api/
│   │   ├── crawler/
│   │   ├── database/
│   │   └── models/
│   ├── candle/                  # Candle 모듈 (DuckDB + Parquet)
│   │   ├── api/
│   │   ├── database/            # DuckDB 쿼리 (Hive 파티션)
│   │   ├── providers/
│   │   └── service/
│   ├── news/                    # News 모듈
│   │   ├── api/
│   │   ├── fetcher/
│   │   ├── pipeline/
│   │   └── store/
│   └── shared/                  # 공유 유틸리티
│       ├── config/
│       └── scheduler/
├── pkg/dart/                    # DART API 클라이언트
├── data/                        # SQLite 파일들
├── storage/                     # DART 문서 저장소
├── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## 라이선스

MIT License
