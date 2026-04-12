# Overnight Web Plan

## Goal

브라우저 + Podman 기반으로 overnight 신호/백테스트 앱을 운영한다.

## Current Scope

- REST API (`/api/*`) 기준으로 프런트/백엔드 통일
- Rust bridge를 통해 신호 생성/백테스트 실행
- Podman compose로 `overnight-app` + `meilisearch` 배포

## Validation Checklist

1. `npm run build` 성공
2. `cargo check --manifest-path src-rust/Cargo.toml --bin overnight-bridge` 성공
3. `npm run start` 후 `/api/health` 정상 응답
4. `npm run e2e` 통과
5. `podman compose -f podman-compose.yaml up -d --build` 성공

## Ops Notes

- 데이터 마운트는 `../backend/data`를 read-only로 유지
- `npm run start`는 빌드된 `src-rust/target/*/overnight-bridge`가 있으면 우선 사용한다
- Podman compose의 Meilisearch는 내부 네트워크 전용이며 호스트 포트를 점유하지 않는다
- 백테스트 성능 튜닝 변수:
  - `ENABLE_BACKTEST_NEWS`
  - `BACKTEST_MAX_CANDIDATES_PER_DAY`
