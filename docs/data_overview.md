# Project Data Overview

Market Mosaic focuses on aggregating financial and corporate data to provide a comprehensive market view.

## 1. DART (Korea Financial Supervisory Service)
- **Corporate Codes**: Full list of corporate codes registered in DART.
- **Corporate Filings**: Real-time and historical corporate filings (reports).
- **Scheduled Job**: `DART-FetchFilings` (Hourly), `DART-DownloadDocs` (Every 5 min).

## 2. Judal (Market Themes & Indicators)
- **Market Themes**: Thematic groups of stocks (e.g., AI, Semiconductors).
- **Stock Indicators**: Detailed stock metrics like PBR, PER, Market Cap, and technical indicators.
- **Historical Data**: Daily snapshots of stock indicators for trend analysis.
- **Scheduled Job**: `Judal-Daily-Crawl` (Daily at 00:00 KST).

## 3. News Ingestion
- **Naver News**: Targeted search for financial/economic news using keywords.
- **NewsAPI**: Global news aggregation (requires API key).
- **Search Engine**: Articles are indexed in Meilisearch for ultra-fast full-text search.
- **Scheduled Job**: `News-Fetch` (Configurable, currently 1 hour).

## 4. Market Prices (OHLCV)
- **Minute/Daily Candles**: Historical data for US (Alpaca) and KR (Kiwoom) markets.
- **Migration API**: Supports batch upsert of external OHLCV data.
