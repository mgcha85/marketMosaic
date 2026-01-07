# DX Unified API 명세서

DX Unified 서버의 API 엔드포인트 명세입니다.

## 기본 정보

- **Base URL**: `http://localhost:8080` (기본값)
- **Content-Type**: `application/json`

---

## 목차

1. [DART (금융감독원 공시)](#1-dart-금융감독원-공시)
2. [Judal (주식 테마/종목)](#2-judal-주식-테마종목)
3. [Candle (캔들 데이터)](#3-candle-캔들-데이터)
4. [News (뉴스)](#4-news-뉴스)

---

## 1. DART (금융감독원 공시)

### 1.1 기업 목록 조회
기업 목록을 페이징하여 조회합니다.

- **Method**: `GET`
- **URL**: `/dart/corps`
- **Query Parameters**:
  - `page` (int, optional): 페이지 번호 (기본값: 1)
  - `limit` (int, optional): 페이지 당 개수 (기본값: 20)

**Response Example:**
```json
{
  "data": [
    {
      "corp_code": "00126380",
      "corp_name": "삼성전자",
      "stock_code": "005930",
      "modified_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1000,
  "page": 1,
  "limit": 20
}
```

### 1.2 공시 목록 조회
공시 제출 목록을 조회합니다. 다양한 필터 조건을 지원합니다.

- **Method**: `GET`
- **URL**: `/dart/filings`
- **Query Parameters**:
  - `page` (int, optional): 페이지 번호 (기본값: 1)
  - `limit` (int, optional): 페이지 당 개수 (기본값: 20)
  - `corp_code` (string, optional): 고유번호
  - `stock_code` (string, optional): 종목코드
  - `date_from` (string, optional): 시작일 (YYYYMMDD)
  - `date_to` (string, optional): 종료일 (YYYYMMDD)

**Response Example:**
```json
{
  "data": [
    {
      "rcept_no": "20240101000001",
      "corp_code": "00126380",
      "corp_name": "삼성전자",
      "report_nm": "주주총회소집공고",
      "rcept_dt": "20240101",
      "flr_nm": "삼성전자",
      "rm": "유",
      "dcm_no": "123456",
      "created_at": "2024-01-01T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

### 1.3 공시 상세 조회
특정 공시의 상세 정보와 관련 문서, 추출된 이벤트를 조회합니다.

- **Method**: `GET`
- **URL**: `/dart/filings/:rcept_no`
- **Path Parameters**:
  - `rcept_no`: 접수번호

**Response Example:**
```json
{
  "filing": {
    "rcept_no": "20240101000001",
    "corp_name": "삼성전자",
    "report_nm": "주주총회소집공고",
    // ... Detail fields
  },
  "documents": [
    {
      "id": 1,
      "rcept_no": "20240101000001",
      "doc_type": "PDF",
      "storage_uri": "s3://...",
      "fetched_at": "2024-01-01T10:05:00Z"
    }
  ],
  "events": [
    {
      "id": 10,
      "event_type": "meeting_schedule",
      "payload_json": "{\"date\": \"2024-03-20\"}",
      "created_at": "2024-01-01T10:10:00Z"
    }
  ]
}
```

---

## 2. Judal (주식 테마/종목)

### 2.1 전체 테마 목록 조회
DB에 저장된 전체 테마 목록을 조회합니다.

- **Method**: `GET`
- **URL**: `/judal/themes`

**Response Example:**
```json
{
  "count": 10,
  "themes": [
    {
      "id": 1,
      "theme_idx": 100,
      "name": "2차전지",
      "stock_count": 50,
      "created_at": "2024-01-01T09:00:00Z",
      "updated_at": "2024-01-01T09:00:00Z"
    }
  ]
}
```

### 2.2 테마 상세 조회
특정 테마의 정보를 조회합니다.

- **Method**: `GET`
- **URL**: `/judal/themes/:themeIdx`
- **Path Parameters**:
  - `themeIdx`: 테마 인덱스

### 2.3 테마별 종목 조회
특정 테마에 속한 종목 목록을 조회합니다.

- **Method**: `GET`
- **URL**: `/judal/themes/:themeIdx/stocks`
- **Path Parameters**:
  - `themeIdx`: 테마 인덱스

**Response Example:**
```json
{
  "theme": { "theme_idx": 100, "name": "2차전지", ... },
  "count": 5,
  "stocks": [
    {
      "id": 1,
      "code": "005930",
      "name": "삼성전자",
      "current_price": 70000,
      "change_rate": 1.5,
      // ... stock fields
    }
  ]
}
```

### 2.4 종목 목록 조회
저장된 종목 목록을 페이징 및 정렬하여 조회합니다.

- **Method**: `GET`
- **URL**: `/judal/stocks`
- **Query Parameters**:
  - `page`, `limit`
  - `sort`: 정렬 필드 (예: `market_cap`, `change_rate`)
  - `order`: `asc` 또는 `desc`
  - `market`: `KOSPI` 또는 `KOSDAQ`

### 2.5 종목 상세 조회
특정 종목의 상세 정보와 관련 테마를 조회합니다.

- **Method**: `GET`
- **URL**: `/judal/stocks/:code`

**Response Example:**
```json
{
  "id": 1,
  "code": "005930",
  "name": "삼성전자",
  "related_themes": ["반도체", "IT"],
  "per": 10.5,
  "pbr": 1.2,
  // ...
}
```

### 2.6 실시간 크롤링 (테마/종목)
Judal 사이트에서 실시간으로 데이터를 크롤링하여 반환합니다.

- **Method**: `GET`
- **URL**: `/judal/realtime/themes/:tab`
- **URL**: `/judal/realtime/stocks/:tab`
- **Path Parameters**:
  - `tab`: 탭 이름 (예: `rising`, `falling`, `hot` 등, `/judal/realtime/tabs`에서 확인 가능)

### 2.7 크롤링 트리거
- **POST** `/judal/crawl`: 전체 크롤링 시작
- **POST** `/judal/crawl/batch`: 일배치 크롤링 시작 (히스토리 저장)

### 2.8 종목 히스토리 조회
- **Method**: `GET`
- **URL**: `/judal/stocks/:code/history`
- **Query Parameters**:
  - `limit`: 개수 (default: 30)

---

## 3. Candle (캔들 데이터)

### 3.1 유니버스 목록 조회
수집 대상 종목(Instruments) 목록을 조회합니다.

- **Method**: `GET`
- **URL**: `/candle/universe`
- **Query Parameters**:
  - `market`: `KR` 또는 `US`
  - `limit`: 개수

### 3.2 캔들 데이터 조회 (전체/검색)
DuckDB를 통해 Parquet 파일의 캔들 데이터를 조회합니다.

- **Method**: `GET`
- **URL**: `/candle/stocks`
- **Query Parameters**:
  - `market`: `KR` (필수)
  - `symbol`: 종목코드
  - `timeframe`: `1m`, `1d` 등 (default: `1m`)
  - `date_from`: 시작 Timestamp
  - `date_to`: 종료 Timestamp
  - `limit`: 개수

**Response Example:**
```json
{
  "count": 100,
  "timeframe": "1m",
  "candles": [
    {
      "market": "KR",
      "symbol": "005930",
      "timestamp": 1704067200,
      "open": 70000,
      "high": 70500,
      "low": 69500,
      "close": 70200,
      "volume": 1000
    }
  ]
}
```

### 3.3 특정 종목 캔들 조회
- **Method**: `GET`
- **URL**: `/candle/stocks/:symbol`

---

## 4. News (뉴스)

### 4.1 뉴스 기사 목록 조회
Meilisearch에 저장된 뉴스를 조회합니다.

- **Method**: `GET`
- **URL**: `/news/articles`
- **Query Parameters**:
  - `limit`, `offset`
  - `source`: 뉴스 소스 (예: `naver`, `newsapi`)
  - `keyword`: 필터링 키워드

### 4.2 뉴스 검색
키워드로 뉴스를 전문 검색(Full-text search)합니다.

- **Method**: `GET`
- **URL**: `/news/search`
- **Query Parameters**:
  - `q`: 검색어 (필수)
  - `limit`: 개수

**Response Example:**
```json
{
  "query": "삼성전자",
  "total": 120,
  "count": 20,
  "articles": [
    {
      "id": "...",
      "title": "삼성전자, 실적 발표",
      "content": "...",
      "published_at": "2024-01-01T10:00:00Z",
      "source": "naver",
      "url": "https://..."
    }
  ]
}
```
