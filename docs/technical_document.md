# Technical Document

## Architecture
The system follows a unified backend approach with a separate frontend.

### Backend (dx-unified)
- **Language**: Go
- **Framework**: Gin (HTTP), GORM (Database)
- **Databases**:
  - **SQLite**: Used for DART filings and Judal theme/stock data.
  - **DuckDB**: Used for high-speed analysis of Candle (OHLCV) data stored in Parquet/Hive format.
- **Search Engine**: Meilisearch for news indexing and full-text search.
- **Scheduler**: Custom internal scheduler for periodic data ingestion.

### Frontend
- **Framework**: Svelte
- **Styling**: Tailwind CSS, DaisyUI
- **Server**: Nginx (serving static files and proxying API requests)

## Deployment
- **Containerization**: Podman-compose
- **CI/CD**: GitHub Actions
- **Auto-start**: Configured via systemd (user-level)

## Configuration
- Environment variables in `.env`
- Overrides allowed via `data/config.json` (managed via Admin Dashboard)
