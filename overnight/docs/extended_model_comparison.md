# Extended Feature Pipeline — 모델 비교 리포트

## 실행 설정
- DB: /mnt/data/projects/kiwoom_restapi/backend/sqlite3/candle_data.db
- 평가 기간: 2021 ~ 2026
- min_gap: 0.005 (0.5%)
- vol_multiplier: 3.0
- threshold: 0.5
- max_positions/day: 2

## 피처 구성

### 기존(Baseline) 피처 (21개)
```
volume, prev_volume, vol_ma60, prev_vol_ma60, ma5, ma50, ma100, ma200, ma400, dist_ma5, dist_ma50, dist_ma100, dist_ma200, dist_ma400, rsi14, ret_1d, ret_5d, ret_20d, intraday_range, intraday_return, market_cap
```

### 확장(Extended) 피처 (34개)
```
volume, prev_volume, vol_ma60, prev_vol_ma60, ma5, ma50, ma100, ma200, ma400, dist_ma5, dist_ma50, dist_ma100, dist_ma200, dist_ma400, rsi14, ret_1d, ret_5d, ret_20d, intraday_range, intraday_return, market_cap, per, pbr, bps, eps, div, dps, log_market_cap, turnover_rate, log_trade_value, sector_mktcap_rank, sector_mktcap_pct, sector_turn_rank, is_kospi
```

**추가된 피처:**
- `per`, `pbr`, `bps`, `eps`, `div`, `dps` — fundamental.db (일별 forward-fill)
- `log_market_cap` — log(시가총액), trade_amount.db
- `turnover_rate` — 거래대금 / 시가총액
- `log_trade_value` — log(거래대금)
- `sector_mktcap_rank` — 섹터 내 시총 순위 (일별, 1=최대)
- `sector_mktcap_pct` — 섹터 내 시총 백분위 (일별)
- `sector_turn_rank` — 섹터 내 거래대금 순위 (일별)
- `is_kospi` — 1=KOSPI, 0=KOSDAQ

## 차트

### 연도별 모델 성과 비교
![연도별 비교](model_comparison_yearly.png)

### 전체 Sum Return 비교
![전체 비교](total_return_comparison.png)


## 전체 성과 요약 (2021–2026)

| model | base_selected | base_prec | base_sum | ext_selected | ext_prec | ext_sum | delta_sum |
|---|---|---|---|---|---|---|---|
| mlp | 1855 | 0.6303 | 54.31 | 1795 | 0.6496 | 55.77 | +1.46 |
| lgbm | 1755 | 0.6531 | 54.59 | 1745 | 0.6568 | 55.23 | +0.64 |
| xgb | 1712 | 0.6646 | 56.85 | 1722 | 0.6618 | 55.91 | -0.94 |
| catboost | 1824 | 0.6425 | 55.32 | 1802 | 0.6489 | 55.51 | +0.19 |


## 모델별 walk-forward 상세

### MLP

**Baseline (기존 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 427 | 0.5363 | 0.0334 | 7.2740 |
| 2022 | 61635 | 243 | 0.5967 | 0.0404 | 6.1067 |
| 2023 | 70449 | 473 | 0.5349 | 0.0376 | 8.8862 |
| 2024 | 80148 | 285 | 0.6561 | 0.0619 | 10.8791 |
| 2025 | 89212 | 300 | 0.7333 | 0.0876 | 15.7017 |
| 2026 | 99290 | 127 | 0.7244 | 0.0858 | 5.4590 |

**Extended (확장 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 490 | 0.5429 | 0.0292 | 7.1689 |
| 2022 | 61635 | 274 | 0.5620 | 0.0325 | 5.1423 |
| 2023 | 70449 | 337 | 0.6172 | 0.0556 | 10.3975 |
| 2024 | 80148 | 262 | 0.6870 | 0.0637 | 11.2916 |
| 2025 | 89212 | 305 | 0.7639 | 0.0883 | 16.5954 |
| 2026 | 99291 | 127 | 0.7244 | 0.0814 | 5.1749 |

> Baseline sum_return: **54.3069** | Extended sum_return: **55.7706** | Delta: **+1.4637**

### LGBM

**Baseline (기존 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 409 | 0.5452 | 0.0333 | 7.2124 |
| 2022 | 61635 | 208 | 0.6202 | 0.0453 | 6.0695 |
| 2023 | 70449 | 468 | 0.5662 | 0.0394 | 9.2827 |
| 2024 | 80148 | 264 | 0.6932 | 0.0695 | 11.6335 |
| 2025 | 89212 | 280 | 0.7714 | 0.0941 | 15.6372 |
| 2026 | 99290 | 126 | 0.7222 | 0.0772 | 4.7520 |

**Extended (확장 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 491 | 0.5540 | 0.0287 | 7.0245 |
| 2022 | 61635 | 275 | 0.5564 | 0.0328 | 5.2524 |
| 2023 | 70449 | 316 | 0.6297 | 0.0569 | 9.9081 |
| 2024 | 80148 | 245 | 0.7265 | 0.0669 | 11.1742 |
| 2025 | 89212 | 291 | 0.7732 | 0.0941 | 16.8705 |
| 2026 | 99291 | 127 | 0.7008 | 0.0780 | 4.9955 |

> Baseline sum_return: **54.5873** | Extended sum_return: **55.2253** | Delta: **+0.6379**

### XGB

**Baseline (기존 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 391 | 0.5550 | 0.0348 | 7.4485 |
| 2022 | 61635 | 209 | 0.6268 | 0.0470 | 6.1874 |
| 2023 | 70449 | 460 | 0.5500 | 0.0412 | 9.7217 |
| 2024 | 80148 | 256 | 0.7031 | 0.0712 | 11.5334 |
| 2025 | 89212 | 271 | 0.8007 | 0.1006 | 16.7348 |
| 2026 | 99290 | 125 | 0.7520 | 0.0811 | 5.2202 |

**Extended (확장 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 491 | 0.5479 | 0.0291 | 7.1672 |
| 2022 | 61635 | 270 | 0.5741 | 0.0350 | 5.3494 |
| 2023 | 70449 | 310 | 0.6290 | 0.0568 | 10.0733 |
| 2024 | 80148 | 240 | 0.7292 | 0.0684 | 11.4126 |
| 2025 | 89212 | 283 | 0.7951 | 0.0961 | 16.9460 |
| 2026 | 99291 | 128 | 0.6953 | 0.0771 | 4.9568 |

> Baseline sum_return: **56.8459** | Extended sum_return: **55.9052** | Delta: **-0.9407**

### CATBOOST

**Baseline (기존 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 421 | 0.5392 | 0.0329 | 7.1697 |
| 2022 | 61635 | 229 | 0.5939 | 0.0417 | 6.0010 |
| 2023 | 70449 | 473 | 0.5518 | 0.0409 | 9.7209 |
| 2024 | 80148 | 285 | 0.6702 | 0.0626 | 10.9757 |
| 2025 | 89212 | 291 | 0.7560 | 0.0917 | 16.2363 |
| 2026 | 99290 | 125 | 0.7440 | 0.0832 | 5.2137 |

**Extended (확장 피처)**

| eval_year | train_rows | selected | precision | avg_return | sum_return |
|---|---|---|---|---|---|
| 2021 | 51889 | 492 | 0.5691 | 0.0290 | 7.1478 |
| 2022 | 61635 | 284 | 0.5493 | 0.0315 | 4.8840 |
| 2023 | 70449 | 332 | 0.6145 | 0.0539 | 10.2444 |
| 2024 | 80148 | 256 | 0.6914 | 0.0665 | 11.6477 |
| 2025 | 89212 | 309 | 0.7638 | 0.0883 | 16.6633 |
| 2026 | 99291 | 129 | 0.7054 | 0.0756 | 4.9215 |

> Baseline sum_return: **55.3173** | Extended sum_return: **55.5088** | Delta: **+0.1916**
