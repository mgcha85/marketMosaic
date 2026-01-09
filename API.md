# Data Migration APIs

This document describes the REST APIs available for migrating data into the system. These endpoints support batch **Upsert** operations, meaning existing records with the same primary key will be overwritten.

> [!WARNING]
> These endpoints do NOT have authentication enabled. Ensure they are used in a secure environment.

---

## 1. Candle Data (US/KR/Crypto)
Ingest historical OHLCV data.

- **Endpoint**: `POST /candle/data`
- **Content-Type**: `application/json`

### Payload Schema
Array of Candle objects:
```json
[
  {
    "market": "US",          // "US", "KR", "CRYPTO"
    "symbol": "AAPL",
    "ts": 1704067200,        // Unix Timestamp (seconds)
    "open": 190.5,
    "high": 192.0,
    "low": 189.5,
    "close": 191.0,
    "volume": 1000000,
    "trade_count": 5000,
    "vwap": 190.8
  }
]
```

### Curl Example
```bash
curl -X POST http://localhost:8080/candle/data \
  -H "Content-Type: application/json" \
  -d '[
    {"market":"US","symbol":"AAPL","ts":1704067200,"open":150,"high":155,"low":149,"close":152,"volume":1000}
  ]'
```

---

## 2. News Articles
Ingest news articles into the search engine (Meilisearch).

- **Endpoint**: `POST /news/migration`
- **Content-Type**: `application/json`

### Payload Schema
Array of Article objects:
```json
[
  {
    "id": "unique_article_id", // Required (e.g., hash of URL)
    "title": "Market hits all-time high",
    "summary": "Stocks rallied today...",
    "url": "https://example.com/news/1",
    "source": "Bloomberg",
    "published_at": "2024-01-01T10:00:00Z"
  }
]
```

### Curl Example
```bash
curl -X POST http://localhost:8080/news/migration \
  -H "Content-Type: application/json" \
  -d '[
    {"id":"abc-123","title":"News Title","url":"http://test.com","published_at":"2024-01-01T00:00:00Z"}
  ]'
```

---

## 3. DART Filings
Ingest corporate filings data.

- **Endpoint**: `POST /dart/migration/filings`
- **Content-Type**: `application/json`

### Payload Schema
Array of Filing objects:
```json
[
  {
    "rcept_no": "20240101000001", // Primary Key
    "corp_code": "00126380",
    "corp_name": "Samsung Electronics",
    "report_nm": "Quarterly Report",
    "rcept_dt": "20240101",
    "flr_nm": "Samsung Electronics",
    "rm": "K"
  }
]
```

### Curl Example
```bash
curl -X POST http://localhost:8080/dart/migration/filings \
  -H "Content-Type: application/json" \
  -d '[
    {"rcept_no":"20240101999999","corp_name":"Test Corp","rcept_dt":"20240101"}
  ]'
```

---

## 4. Judal (Themes & Stocks)
Ingest theme and stock metadata.

### 4.1 Themes
- **Endpoint**: `POST /judal/migration/themes`
- **Content-Type**: `application/json`

#### Payload Schema
```json
[
  {
    "theme_idx": 101, // Primary Key
    "name": "Semiconductors",
    "stock_count": 50
  }
]
```

### 4.2 Stocks
- **Endpoint**: `POST /judal/migration/stocks`
- **Content-Type**: `application/json`

#### Payload Schema
```json
[
  {
    "code": "005930", // Primary Key
    "name": "Samsung Electronics",
    "market": "KOSPI",
    "current_price": 75000,
    "market_cap": 450000000000000
  }
]
```

### Curl Example
```bash
curl -X POST http://localhost:8080/judal/migration/themes \
  -H "Content-Type: application/json" \
  -d '[{"theme_idx":1,"name":"Test Theme","stock_count":10}]'
```
