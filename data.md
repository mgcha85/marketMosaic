# 데이터 위치 및 사용법 가이드

> 현재까지 누적된 marketMosaic 프로젝트의 데이터 현황, 저장 위치, 접근 방법을 정리한 문서입니다.  
> **최종 업데이트: 2026-04-11**

---

## 📊 데이터 누적 현황 (요약)

| 항목 | Judal 주식 | News 뉴스 | DART 공시 |
|-----|----------|---------|---------|
| **저장소** | SQLite | Meilisearch | SQLite |
| **데이터 개수** | 40,086 (히스토리) | 136,009 (기사) | 27 MB (파일 크기) |
| **시작일** | 2026-01-06 | 2026-03-14 | - |
| **최신 데이터** | 2026-04-11 | 2026-04-11 | 2026-04-10 23:13 |
| **누적 기간** | 95일 | 28일 | - |
| **상세 날짜** | 12개 날짜 스냅샷 | - | - |

---

## 🗂️ 데이터 저장 위치

### 1. **Judal - 주식 데이터** (SQLite)

**파일 위치:**
```
backend/data/judal.db (8.0 MB)
```

**테이블 구조:**
```sql
-- 테마 정보
themes
├── id (PK)
├── theme_idx (유니크)
├── name
├── stock_count
└── created_at, updated_at

-- 현재 및 최신 종목 정보
stocks
├── id (PK)
├── code (유니크) - 종목 코드
├── name, market
├── current_price, price_change, change_rate
├── pbr, per, eps, market_cap
├── 52주 고가/저가, 예상수익률 등

-- 일별 종목 스냅샷
stock_history
├── id (PK)
├── crawl_date (DATE)
├── code, name, market
├── 모든 가격/지표 정보
└── UNIQUE(crawl_date, code)

-- 테마-종목 매핑
theme_stocks
├── theme_idx
└── stock_code

-- 크롤링 로그
crawl_logs
├── crawl_date
├── crawl_type
├── themes_count, stocks_count
└── duration_seconds, status
```

**데이터 현황:**
- 테마: 322개
- 종목: 3,440개
- 종목 히스토리: 40,086개 (12개 날짜)
- 테마-종목 매핑: 6,844개

**API 사용법:**

```bash
# 1. 테마 목록 조회
curl http://localhost:8080/judal/themes

# 2. 특정 테마의 종목 조회
curl http://localhost:8080/judal/themes/214/stocks

# 3. 종목 목록 (페이지네이션)
curl "http://localhost:8080/judal/stocks?limit=20&sort=market_cap&order=desc"

# 4. 특정 종목 상세 정보
curl http://localhost:8080/judal/stocks/005930

# 5. 특정 종목의 히스토리 (일별 스냅샷)
curl http://localhost:8080/judal/stocks/005930/history

# 6. 크롤러 상태 확인
curl http://localhost:8080/judal/status

# 7. 실시간 탭 데이터 (크롤링)
curl http://localhost:8080/judal/realtime/themes/rising
curl http://localhost:8080/judal/realtime/stocks/rising
```

**직접 DB 접근:**

```bash
# DB 조회 예시 (Python)
import sqlite3

db = sqlite3.connect('backend/data/judal.db')
cursor = db.cursor()

# 최신 종목 100개 (시가총액 기준)
cursor.execute("""
    SELECT code, name, current_price, market_cap 
    FROM stocks 
    ORDER BY market_cap DESC 
    LIMIT 100
""")

# 특정 날짜의 종목 히스토리
cursor.execute("""
    SELECT code, name, current_price, change_rate 
    FROM stock_history 
    WHERE crawl_date = '2026-04-11'
""")

db.close()
```

---

### 2. **NEWS - 뉴스 데이터** (Meilisearch)

**저장소 위치:**
```
Meilisearch 서버
├── 포트: 7700 (내부), localhost:7700 (호스트)
├── 인덱스: articles
└── 인증: Bearer masterKey
```

**인덱스 구조:**
```
articles (136,009개 문서)
├── id (Primary Key)
├── title
├── summary (description)
├── url
├── canonical_url
├── source (e.g., "naver_search_api")
├── publisher
├── published_at (ISO 8601)
├── fetched_at (ISO 8601)
└── dup_state (중복 상태)
```

**데이터 현황:**
- 총 기사: 136,009개
- 가장 오래된 기사: 2026-03-14 18:39:57 (UTC)
- 가장 최신 기사: 2026-04-11 00:55:00 (+09:00)
- 누적 기간: 28일

**API 사용법:**

```bash
# 1. 뉴스 검색 (기본)
curl "http://localhost:8080/news/search?q=삼성전자&limit=10"

# 2. 페이지네이션
curl "http://localhost:8080/news/search?q=&limit=50&offset=0"

# 3. 전체 뉴스 조회 (limit=1000)
curl "http://localhost:8080/news/search?q=&limit=1000"
```

**Meilisearch 직접 접근:**

```bash
# 1. 기사 검색
curl -X POST http://localhost:7700/indexes/articles/search \
  -H "Authorization: Bearer masterKey" \
  -H "Content-Type: application/json" \
  -d '{
    "q": "뉴스 검색어",
    "limit": 20,
    "sort": ["published_at:desc"]
  }'

# 2. 가장 오래된 기사 확인
curl -X POST http://localhost:7700/indexes/articles/search \
  -H "Authorization: Bearer masterKey" \
  -H "Content-Type: application/json" \
  -d '{"limit": 1, "sort": ["published_at:asc"]}'

# 3. 가장 최신 기사 확인
curl -X POST http://localhost:7700/indexes/articles/search \
  -H "Authorization: Bearer masterKey" \
  -H "Content-Type: application/json" \
  -d '{"limit": 1, "sort": ["published_at:desc"]}'

# 4. 인덱스 통계
curl -H "Authorization: Bearer masterKey" \
  http://localhost:7700/indexes/articles/stats
```

**Python 예시:**

```python
import requests
import json

# Meilisearch 직접 검색
headers = {"Authorization": "Bearer masterKey"}

# 검색
response = requests.post(
    "http://localhost:7700/indexes/articles/search",
    headers=headers,
    json={
        "q": "금리 인상",
        "limit": 100,
        "sort": ["published_at:desc"]
    }
)

articles = response.json()['hits']

for article in articles:
    print(f"{article['published_at']}: {article['title']}")
```

---

### 3. **DART - 공시 데이터** (SQLite)

**파일 위치:**
```
backend/data/dart.db (27 MB)
```

**특징:**
- 한국 금융감독원 공시 정보 저장
- 상세 구조는 `.agents/AGENTS.md` 참고
- API: `/dart/*` 엔드포인트

**API 사용법:**

```bash
# 1. 기업 목록
curl "http://localhost:8080/dart/corps?page=1&limit=20"

# 2. 공시 목록
curl "http://localhost:8080/dart/filings?corp_code=00370&date_from=2026-01-01"

# 3. 공시 상세
curl http://localhost:8080/dart/filings/20260101000001
```

---

## 🔌 API 서버 설정

### 포트 및 엔드포인트

```
┌─────────────────────────────────────────────────────────┐
│ 서비스                                                  │
├─────────────────────────────────────────────────────────┤
│ 백엔드 API 서버      : http://localhost:8080            │
│ 프론트엔드 (Nginx)   : http://localhost:8090            │
│ Meilisearch         : http://localhost:7700            │
│ Vite Dev Server     : http://localhost:5173 (개발 시)  │
└─────────────────────────────────────────────────────────┘
```

### 헬스체크

```bash
curl http://localhost:8080/health
# 응답: {"service":"dx-unified","status":"ok"}
```

---

## 🛠️ 데이터베이스 직접 접근

### SQLite 조회 (Python)

```python
import sqlite3

# Judal DB
judal_db = sqlite3.connect('backend/data/judal.db')
judal_cursor = judal_db.cursor()

# DART DB
dart_db = sqlite3.connect('backend/data/dart.db')
dart_cursor = dart_db.cursor()

# 예: 가장 큰 시가총액 10개 종목
judal_cursor.execute("""
    SELECT code, name, market_cap 
    FROM stocks 
    WHERE market_cap IS NOT NULL 
    ORDER BY market_cap DESC 
    LIMIT 10
""")

for code, name, market_cap in judal_cursor.fetchall():
    print(f"{code:6} | {name:20} | {market_cap:,}")

judal_db.close()
dart_db.close()
```

### Meilisearch 통계

```bash
# 문서 개수
curl -H "Authorization: Bearer masterKey" \
  http://localhost:7700/indexes/articles/stats | jq .numberOfDocuments
# 응답: 136009
```

---

## 📈 데이터 스냅샷 분석

### Judal 히스토리 분석

```bash
# Python으로 일별 변동 추적
python3 << 'EOF'
import sqlite3
from datetime import datetime

db = sqlite3.connect('backend/data/judal.db')
cursor = db.cursor()

# 각 날짜별 데이터 개수
cursor.execute("""
    SELECT crawl_date, COUNT(*) as stock_count 
    FROM stock_history 
    GROUP BY crawl_date 
    ORDER BY crawl_date
""")

for date, count in cursor.fetchall():
    print(f"{date}: {count}개 종목")

db.close()
EOF
```

---

## 💾 데이터 유지보수

### 백업

```bash
# Judal DB 백업
cp backend/data/judal.db backend/data/judal.db.backup

# DART DB 백업
cp backend/data/dart.db backend/data/dart.db.backup

# Meilisearch 데이터 (Docker 볼륨)
docker-compose exec meilisearch tar -czf /meili_data/backup.tar.gz /meili_data
```

### 크롤링 상태 확인

```bash
# Judal 크롤러 상태
curl http://localhost:8080/judal/status | jq .

# 크롤링 실행 로그
curl http://localhost:8080/judal/runs | jq .
```

---

## 🚀 데이터 사용 시나리오

### 1. 상승 테마의 종목 조회

```bash
# 실시간 상승 테마
curl http://localhost:8080/judal/realtime/themes/rising | jq '.[] | {theme_idx, name, stock_count}'

# 해당 테마의 종목
curl http://localhost:8080/judal/themes/214/stocks | jq '.[] | {code, name, change_rate}'
```

### 2. 특정 기간의 종목 가격 변동 추적

```python
import sqlite3

db = sqlite3.connect('backend/data/judal.db')
cursor = db.cursor()

# 삼성전자의 일별 가격 변동
cursor.execute("""
    SELECT crawl_date, current_price, change_rate 
    FROM stock_history 
    WHERE code = '005930' 
    ORDER BY crawl_date
""")

for date, price, change in cursor.fetchall():
    print(f"{date}: {price:,}원 ({change:+.2f}%)")

db.close()
```

### 3. 최신 재정 관련 뉴스 조회

```bash
curl -X POST http://localhost:7700/indexes/articles/search \
  -H "Authorization: Bearer masterKey" \
  -H "Content-Type: application/json" \
  -d '{
    "q": "금리 금융",
    "limit": 20,
    "sort": ["published_at:desc"]
  }' | jq '.hits[] | {published_at, title, source}'
```

---

## ⚠️ 주의사항

### DuckDB (Candle 데이터)

```
위치: backend/data/candles/market={KR|US}/year=YYYY/month=MM/*.parquet
특징: Hive 파티션 구조, 읽기 전용 분석 DB
```

- **절대 쓰기 금지**: DuckDB는 읽기 전용입니다
- **파티션 필터 필수**: 쿼리 성능을 위해 market, year, month 필터 적용

### 민감 정보

```
❌ .env 파일 직접 수정 금지
❌ API 키/시크릿 로그 출력 금지
❌ 데이터베이스 파일 직접 편집 금지
```

---

## 📞 더 알아보기

- **아키텍처 상세:** `docs/technical_document.md`
- **에이전트 지침:** `.agents/AGENTS.md`
- **데이터 개요:** `docs/data_overview.md`
- **API 문서:** `API.md`

