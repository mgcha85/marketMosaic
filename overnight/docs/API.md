# Overnight REST API

Frontend(Svelte)와 Backend(Node + Rust bridge) 사이의 표준 인터페이스는 HTTP REST다.

## Base

- Local: `http://127.0.0.1:1420`
- Prefix: `/api`

## System

- `GET /api/health`
- `GET /api/app-paths`
- `GET /api/available-dates`

## Signals

- `GET /api/signals?date=YYYY-MM-DD`
- `POST /api/signals/generate`
- `GET /api/signals/:code?date=YYYY-MM-DD`

### POST /api/signals/generate body

```json
{
  "date": "2026-04-11",
  "minScore": 8.0
}
```

## Backtests

- `POST /api/backtests/start`
- `GET /api/backtests/jobs`
- `GET /api/backtests/jobs/:jobId`
- `GET /api/backtests/runs`
- `GET /api/backtests/stats`
- `GET /api/backtests/:runId`
- `GET /api/backtests/:runId/day-review?date=YYYY-MM-DD`

### POST /api/backtests/start body

```json
{
  "dateFrom": "2026-01-01",
  "dateTo": "2026-01-31",
  "minScore": 8.0
}
```

## Error format

```json
{
  "error": "message"
}
```
