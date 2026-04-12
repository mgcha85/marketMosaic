# IMPLEMENT.md

> 구현 로그. 작업 중 결정 사항, 이탈 내용, 미해결 문제를 기록합니다.  
> 실시간으로 추가하고, **덮어쓰지 않습니다**. 각 항목에 날짜를 기록합니다.

---

## 작업 참조

**PLAN.md:** `.agents/PLAN.md`  
**작업:** <!-- PLAN.md의 Task 항목 복사 -->  
**시작일:** YYYY-MM-DD

---

## 결정 로그 (Decision Log)

<!-- 형식: ### YYYY-MM-DD: <결정 제목> -->
<!-- 각 결정에: 상황(context) → 선택지(options) → 결정(decision) → 이유(rationale) -->

### YYYY-MM-DD: 초기 접근 방식 선택

**상황:**  

**선택지:**  
1. 
2. 

**결정:**  

**이유:**  

---

## 이탈 사항 (Deviations from Plan)

<!-- 계획에서 변경된 사항과 이유 -->
<!-- 형식: - YYYY-MM-DD: <변경 내용> — 이유: <왜 계획과 달라졌는가> -->

---

## 발견된 문제 (Issues Found)

<!-- 작업 중 발견한 버그, 기술 부채, 예상치 못한 제약 -->
<!-- 이 작업의 범위를 벗어나면 docs/TODO.md에 추가 -->

| 발견일 | 설명 | 심각도 | 처리 방법 |
|--------|------|--------|---------|
| | | low/mid/high | 이 작업에서 수정 / TODO에 추가 / 무시 |

---

## 파일 변경 요약 (Files Changed)

<!-- 마지막에 변경된 파일 목록과 변경 이유 -->

| 파일 | 변경 유형 | 설명 |
|------|---------|------|
| `backend/internal/.../` | 추가/수정/삭제 | |
| `frontend/src/.../` | 추가/수정/삭제 | |

---

## 검증 결과 (Verification Results)

```bash
# go build
$ cd backend && go build ./...
# 결과: 

# go vet
$ cd backend && go vet ./...
# 결과: 

# go test (해당 도메인)
$ cd backend && go test ./internal/...
# 결과: 
```

---

## 미해결 항목 (Open Items)

<!-- 이 작업 완료 후에도 남아 있는 항목 -->
<!-- docs/TODO.md로 이관 대상 -->

- [ ] 

---

*마지막 업데이트: YYYY-MM-DD*
