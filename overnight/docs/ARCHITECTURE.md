# Overnight Web Architecture

## Overview

Overnight는 Podman으로 배포되는 웹 애플리케이션이다.

- Frontend: Svelte 5 + Tailwind + DaisyUI
- API Server: Node.js (`server/index.js`)
- Core Engine: Rust bridge (`overnight-bridge`)
- Storage: SQLite (`overnight.db`)
- Source data: `judal.db`, candles(읽기 전용), Meilisearch

## Runtime Flow

1. 브라우저가 `/api/*` REST 요청을 보낸다.
2. Node 서버가 요청을 받아 Rust bridge 명령을 실행한다.
3. Rust 엔진이 점수/백테스트를 수행하고 JSON 결과를 반환한다.
4. Node 서버가 응답을 JSON으로 전달한다.

## Directory Roles

- `src/`: Svelte UI
- `server/`: Express REST server
- `src-rust/src/`: Rust core modules (`agent`, `backtest`, `database`, `model`, `service`, `bin`)
- `data/`: app DB 저장 경로
- `docs/`: 운영/개발 문서

## Deployment

- Container: `Dockerfile`
- Orchestration: `podman-compose.yaml`
- Environment: `.env.container`
- Data mounts:
  - `./data -> /app/data` (read-write)
  - `../backend/data -> /backend/data` (read-only)
