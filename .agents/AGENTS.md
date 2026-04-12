# AGENTS.md

> AI 에이전트가 이 레포지토리에서 작업하기 전에 반드시 읽어야 할 프로젝트 수준의 지침입니다.

---

## 프로젝트 개요

**marketMosaic**는 한국/미국 금융 데이터를 통합 수집·제공하는 플랫폼입니다.  
Go(Gin) 백엔드가 4개의 독립 도메인(DART 공시, Judal 테마, Candle OHLCV, News 뉴스)을 단일 서버에서 운영하며, Svelte 프론트엔드가 Nginx를 통해 정적 파일과 API 프록시를 제공합니다.

**핵심 기술 스택:**
- Backend: Go 1.22+, Gin, GORM, DuckDB, SQLite, Meilisearch
- Frontend: Svelte, Tailwind CSS, DaisyUI, Vite
- Infra: Docker / Podman-compose, systemd (user-level)

---

## 레포지토리 구조

```
marketMosaic/
├── .agents/              # 에이전트 하네스 문서 (이 폴더)
├── backend/
│   ├── cmd/
│   │   ├── server/main.go      # 메인 서버 엔트리포인트
│   │   └── seeder/main.go      # 데이터 시딩 스크립트
│   ├── internal/
│   │   ├── dart/               # DART 공시 도메인
│   │   │   ├── api/handlers.go
│   │   │   ├── database/
│   │   │   ├── models/
│   │   │   └── scheduler/
│   │   ├── judal/              # 주식 테마/종목 크롤링 도메인
│   │   │   ├── api/handlers.go
│   │   │   ├── config/
│   │   │   ├── crawler/
│   │   │   ├── database/
│   │   │   └── models/
│   │   ├── candle/             # OHLCV 캔들 데이터 도메인
│   │   │   ├── api/handlers.go
│   │   │   ├── database/       # DuckDB 쿼리
│   │   │   ├── model/
│   │   │   ├── providers/      # alpaca, fmp, kiwoom, kiwoomrest
│   │   │   └── service/
│   │   ├── news/               # 뉴스 수집·검색 도메인
│   │   │   ├── api/handlers.go
│   │   │   ├── fetcher/        # naver, newsapi
│   │   │   ├── pipeline/       # filter, normalize, dedup
│   │   │   └── store/meili/
│   │   ├── admin/              # 어드민 대시보드 API
│   │   └── shared/
│   │       ├── config/config.go
│   │       └── scheduler/
│   ├── pkg/dart/client.go
│   ├── data/candles/           # Parquet/Hive 파티션 데이터
│   │   └── market=KR/year=YYYY/month=MM/*.parquet
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── stores.js
│   │   │   └── components/     # AdminPanel, CandleChart, Navbar
│   │   └── App.svelte
│   └── nginx.conf
├── docs/
│   ├── technical_document.md
│   ├── data_overview.md
│   └── TODO.md
├── docker-compose.yml
├── docker-compose.prod.yml
└── .env                        # 민감 정보 (절대 수정 금지)
```

---

## 도메인별 패턴 규칙

각 도메인은 다음 레이어 구조를 따릅니다. 신규 도메인 추가 시 반드시 이 구조를 준수합니다:

```
internal/{domain}/
├── api/handlers.go     # Gin 핸들러, 입력 검증, HTTP 응답만 담당
├── database/           # DB 쿼리/접근 레이어 (비즈니스 로직 없음)
├── model(s)/           # 도메인 모델/타입 정의
└── service/            # 비즈니스 로직 (있는 경우)
```

- **핸들러는 DB를 직접 호출하지 않습니다.** service 또는 database 레이어를 통해야 합니다.
- **에러 응답**은 `gin.Context.JSON(http.StatusXXX, gin.H{"error": "..."})` 형식을 사용합니다.
- **모델 파일**은 Go 구조체 정의만 포함하며, 메서드 로직은 service 레이어에 둡니다.

---

## 코드 컨벤션

### Go 백엔드
- **포매터**: `gofmt` (저장 시 자동 실행 권장)
- **린터**: `go vet ./...`
- **네이밍**: Go 표준 컨벤션 (CamelCase for exported, camelCase for unexported)
- **에러 처리**: `errors.New("...")` 또는 `fmt.Errorf("...: %w", err)` 래핑 방식
- **import 순서**: 표준 라이브러리 → 서드파티 → 내부 패키지 (빈 줄로 구분)
- **패키지당 하나의 책임**: 핸들러 파일은 HTTP 로직만, DB 파일은 쿼리만

### Svelte 프론트엔드
- **포매터**: Prettier (`npx prettier --write`)
- **스타일**: Tailwind CSS 유틸리티 클래스 + DaisyUI 컴포넌트
- **상태 관리**: Svelte stores (`lib/stores.js`)
- **컴포넌트**: `lib/components/` 에 위치, PascalCase 파일명

### 커밋 규칙
```
feat(domain): 기능 설명
fix(domain): 수정 내용
refactor(domain): 리팩토링 내용
docs: 문서 변경
chore: 빌드/설정 변경
```
`domain`은 `dart`, `judal`, `candle`, `news`, `admin`, `frontend`, `infra` 중 하나입니다.

---

## 데이터베이스 주의사항

| DB | 용도 | 위치 |
|----|------|------|
| SQLite | DART 공시, Judal 테마/종목 | `storage/*.db` (런타임 생성) |
| DuckDB | 캔들 OHLCV 분석 쿼리 | `backend/data/candles/` Parquet 파일 직접 쿼리 |
| Meilisearch | 뉴스 전문 검색 | Docker 컨테이너 (포트 7700) |

**Candle 데이터 Hive 파티션 구조:**
```
data/candles/market={KR|US}/year=YYYY/month=MM/*.parquet
```
DuckDB 쿼리 시 반드시 Hive 파티션 필터를 적용해야 성능이 보장됩니다.

**절대 금지:**
- DuckDB에 쓰기 작업 수행 (읽기 전용 분석 DB)
- `.env` 파일 수정 또는 시크릿 값 로그 출력
- `storage/` 폴더의 `.db` 파일 직접 편집

---

## 테스트 실행

> 현재 테스트 커버리지가 낮습니다. 새 기능 추가 시 인접 핸들러나 서비스의 단위 테스트를 함께 작성합니다.

```bash
# 백엔드 전체 테스트
cd backend && go test ./...

# 특정 도메인 테스트
cd backend && go test ./internal/dart/...

# 린트
cd backend && go vet ./...

# 프론트엔드 빌드 확인
cd frontend && npm run build
```

---

## 도구 권한 (Tool Permissions)

### 허용

- `backend/internal/**` 파일 읽기/편집
- `frontend/src/**` 파일 읽기/편집
- `docs/**` 파일 읽기/편집
- `.agents/**` 파일 읽기/편집
- `go test ./...`, `go vet ./...`, `go build ./...` 실행
- `npm run build`, `npm run dev` 실행
- `docker-compose` 상태 확인 (`ps`, `logs`)

### 확인 후 진행

- `backend/cmd/` 메인 엔트리포인트 수정
- `docker-compose.yml`, `docker-compose.prod.yml` 수정
- 새 외부 의존성 추가 (`go get`, `npm install`)
- `backend/internal/shared/config/` 수정 (전체 영향)
- DB 스키마 변경 (마이그레이션 필요)

### 절대 금지

- `.env` 파일 읽기 또는 수정
- `storage/` 내 `.db` 파일 직접 편집/삭제
- `data/candles/` Parquet 파일 수정/삭제
- `git push`, 브랜치 삭제 등 원격 작업
- `rm -rf` 등 파괴적 명령
- 외부 네트워크 요청 (크롤러 직접 실행 포함)

---

## 알려진 제약사항

1. **테스트 없음**: 현재 단위 테스트가 거의 없습니다. 새 기능에는 테스트를 추가하되, 기존 코드의 테스트 부재를 이유로 작업을 블로킹하지 마십시오.
2. **Admin 인증 미구현**: `/admin` 엔드포인트에 인증이 없습니다. Admin API를 수정할 때 인증 없는 상태임을 인지하고 민감 동작을 추가하지 마십시오.
3. **DuckDB 단일 연결**: DuckDB는 파일 잠금으로 인해 동시 쓰기가 불가합니다. Candle 수집과 쿼리가 충돌할 수 있으므로 쓰기 경로를 변경할 때 주의하십시오.
4. **Meilisearch 의존성**: 뉴스 검색 기능은 Meilisearch 컨테이너가 실행 중이어야 합니다. 로컬 개발 시 `docker-compose up meilisearch`가 필요합니다.
5. **Kiwoom API는 Windows 전용**: `providers/kiwoom/` 클라이언트는 키움증권 OpenAPI가 Windows에서만 동작하므로 Linux CI에서는 빌드 태그로 건너뜁니다.

---

## 검증 게이트 (Verification Gates)

작업 완료 전 반드시 확인:

- [ ] `cd backend && go build ./...` 성공
- [ ] `cd backend && go vet ./...` 경고 없음
- [ ] 변경된 도메인의 테스트 통과 (`go test ./internal/{domain}/...`)
- [ ] 새 API 엔드포인트는 `docs/` 또는 `API.md`에 문서화
- [ ] 민감 정보(API 키, 패스워드)가 코드/로그에 포함되지 않음
- [ ] 수정 범위가 허용된 파일 경로 내에 있음

---

## 에스컬레이션

허용 범위 밖의 결정이 필요하거나 막히는 경우, **가정으로 진행하지 말고 블로커를 명확히 설명하고 중단**하십시오.  
특히 다음 상황에서는 반드시 사람의 확인이 필요합니다:
- DB 스키마 파괴적 변경
- 외부 API 크레덴셜 접근이 필요한 작업
- 프로덕션 환경(`docker-compose.prod.yml`) 영향 변경
