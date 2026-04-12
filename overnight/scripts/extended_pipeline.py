#!/usr/bin/env python3
"""
Extended feature walk-forward backtest
- 기존 20개 tabular 피처 + fundamental / trade_amount / meta 확장 피처
- 모델 비교: MLP, LightGBM, XGBoost, CatBoost
- walk-forward expanding window (2021~2026)
- 기존 결과(baseline) 대비 비교 리포트 생성
"""
from __future__ import annotations

import argparse
import sqlite3
import sys
from pathlib import Path

import catboost as cb
import lightgbm as lgb
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np
import polars as pl
import torch
import xgboost as xgb
from sklearn.preprocessing import LabelEncoder, StandardScaler

ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from scripts.overnight_pipeline import (
    Config,
    MLPClassifier,
    aggregate_period,
    build_features,
    feature_columns,
    impute_scale,
    list_symbol_tables,
    load_ohlcv,
    train_classifier_model,
    train_regressor,
)

# ──────────────────────────────────────────────────────────────────────────────
# 추가 DB 로더
# ──────────────────────────────────────────────────────────────────────────────

def load_meta(meta_db: Path) -> pl.DataFrame:
    """meta.db → symbol코드, 업종(sector), 시장구분(market), 지역 매핑"""
    conn = sqlite3.connect(meta_db)
    rows = conn.execute(
        "SELECT 종목코드, 시장구분, 업종, 지역 FROM stockList"
    ).fetchall()
    conn.close()
    df = pl.DataFrame(
        rows,
        schema=["code", "market", "sector", "region"],
        orient="row",
    )
    return df.with_columns(pl.col("code").str.strip_chars())


def load_trade_amount(ta_db: Path, symbols: list[str], since_year: int = 2015) -> pl.DataFrame:
    """trade_amount.db → UNION ALL 청크 방식으로 빠르게 로드"""
    conn = sqlite3.connect(ta_db)
    since_str = f"{since_year}-01-01"
    chunk_size = 50
    frames: list[pl.DataFrame] = []

    for i in range(0, len(symbols), chunk_size):
        chunk = symbols[i : i + chunk_size]
        parts = []
        for sym in chunk:
            escaped = sym.replace('"', '""')
            parts.append(
                f'SELECT \'{sym}\' AS symbol, Date AS date, '
                f'시가총액 AS market_cap, 거래대금 AS trade_value, 상장주식수 AS listed_shares '
                f'FROM "{escaped}" WHERE Date >= \'{since_str}\''
            )
        sql = " UNION ALL ".join(parts)
        try:
            df = pl.read_database(sql, connection=conn)
            if df.height:
                frames.append(df)
        except Exception:
            # 청크 실패 시 개별 fallback
            for sym in chunk:
                try:
                    escaped = sym.replace('"', '""')
                    df2 = pl.read_database(
                        f'SELECT \'{sym}\' AS symbol, Date AS date, '
                        f'시가총액 AS market_cap, 거래대금 AS trade_value, 상장주식수 AS listed_shares '
                        f'FROM "{escaped}" WHERE Date >= \'{since_str}\'',
                        connection=conn,
                    )
                    if df2.height:
                        frames.append(df2)
                except Exception:
                    continue

    conn.close()
    if not frames:
        return pl.DataFrame({"symbol": [], "date": [], "market_cap": [], "trade_value": [], "listed_shares": []})
    out = pl.concat(frames, how="vertical_relaxed")
    return out.with_columns(pl.col("date").str.strptime(pl.Date, strict=False))


def load_fundamental(fund_db: Path, since_year: int = 2015) -> pl.DataFrame:
    """
    fundamental.db: 날짜 테이블(YYYYMMDD) → long 포맷 변환
    since_year 이후 날짜만 로드 (속도 최적화).
    """
    conn = sqlite3.connect(fund_db)
    tables = sorted([
        t[0]
        for t in conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
        ).fetchall()
        if t[0].isdigit() and len(t[0]) == 8
    ])
    # since_year 이후만
    tables = [t for t in tables if int(t[:4]) >= since_year]

    # pandas read_sql로 한 번에 읽기 (UNION ALL 청크)
    chunk_size = 100
    frames: list[pl.DataFrame] = []
    for i in range(0, len(tables), chunk_size):
        chunk = tables[i : i + chunk_size]
        parts = [
            f'SELECT \'{t}\' AS date_str, 티커 AS code, BPS, PER, PBR, EPS, DIV, DPS FROM "{t}"'
            for t in chunk
        ]
        sql = " UNION ALL ".join(parts)
        try:
            df = pl.read_database(sql, connection=conn)
            frames.append(df)
        except Exception:
            continue
    conn.close()

    if not frames:
        return pl.DataFrame()

    out = pl.concat(frames, how="vertical_relaxed")
    out = out.with_columns(
        pl.col("date_str").str.strptime(pl.Date, format="%Y%m%d", strict=False).alias("date")
    ).drop("date_str")
    return out.with_columns(pl.col("code").str.strip_chars())


# ──────────────────────────────────────────────────────────────────────────────
# 확장 피처 빌더
# ──────────────────────────────────────────────────────────────────────────────

SECTOR_COL = "sector"
EXTRA_FEATURE_COLS = [
    # fundamental (forward-filled daily)
    "per", "pbr", "bps", "eps", "div", "dps",
    # trade_amount (daily)
    "log_market_cap",
    "turnover_rate",       # 거래대금 / 시가총액
    "log_trade_value",
    # sector-relative
    "sector_mktcap_rank",  # 섹터 내 시총 순위 (작을수록 상위)
    "sector_mktcap_pct",   # 섹터 내 시총 백분위 (0~1, 높을수록 상위)
    "sector_turn_rank",    # 섹터 내 거래대금 순위
    # market type
    "is_kospi",            # 1=KOSPI, 0=KOSDAQ
]


def build_extended_features(
    candle_df: pl.DataFrame,
    cfg: Config,
    meta_df: pl.DataFrame,
    ta_df: pl.DataFrame,
    fund_df: pl.DataFrame,
) -> pl.DataFrame:
    """기존 build_features() 결과에 외부 DB 피처를 join/계산하여 결합."""
    # 기존 피처 생성
    base = build_features(candle_df, cfg, metadata=None)

    # ── 심볼 코드 추출 (000020.KS → 000020) ──────────────────────────────────
    base = base.with_columns(
        pl.col("symbol").str.split(".").list.first().alias("code")
    )

    # ── meta join: sector, market, region ────────────────────────────────────
    if meta_df.height:
        base = base.join(meta_df, on="code", how="left")
    else:
        base = base.with_columns([
            pl.lit(None, dtype=pl.Utf8).alias("market"),
            pl.lit(None, dtype=pl.Utf8).alias("sector"),
            pl.lit(None, dtype=pl.Utf8).alias("region"),
        ])

    base = base.with_columns(
        (pl.col("market") == "코스피").cast(pl.Int8).alias("is_kospi")
    )

    # ── trade_amount join: 시가총액, 거래대금 ─────────────────────────────────
    if ta_df.height:
        base = base.join(ta_df, on=["symbol", "date"], how="left")
    else:
        base = base.with_columns([
            pl.lit(None, dtype=pl.Float64).alias("market_cap"),
            pl.lit(None, dtype=pl.Float64).alias("trade_value"),
            pl.lit(None, dtype=pl.Float64).alias("listed_shares"),
        ])

    base = base.with_columns([
        pl.col("market_cap").cast(pl.Float64).fill_null(0.0),
        pl.col("trade_value").cast(pl.Float64).fill_null(0.0),
    ])
    base = base.with_columns([
        pl.when(pl.col("market_cap") > 0)
          .then(pl.col("market_cap").log())
          .otherwise(0.0)
          .alias("log_market_cap"),
        pl.when(pl.col("market_cap") > 0)
          .then(pl.col("trade_value") / pl.col("market_cap"))
          .otherwise(0.0)
          .alias("turnover_rate"),
        pl.when(pl.col("trade_value") > 0)
          .then(pl.col("trade_value").log())
          .otherwise(0.0)
          .alias("log_trade_value"),
    ])

    # ── 섹터 내 시총 순위 (일별) ───────────────────────────────────────────────
    # rank: 같은 날짜 + 섹터 내에서 시총 내림차순 순위 (1 = 가장 큰 회사)
    base = base.with_columns([
        pl.col("market_cap")
          .rank(method="ordinal", descending=True)
          .over(["date", "sector"])
          .alias("sector_mktcap_rank"),
        # 백분위: (rank - 1) / (count - 1), 1 = 상위, 0 = 하위
        (
            (pl.col("market_cap").rank(method="ordinal", descending=True).over(["date", "sector"]) - 1)
            / (pl.col("market_cap").count().over(["date", "sector"]) - 1).clip(lower_bound=1)
        ).alias("sector_mktcap_pct"),
        pl.col("trade_value")
          .rank(method="ordinal", descending=True)
          .over(["date", "sector"])
          .alias("sector_turn_rank"),
    ])

    # ── fundamental join (forward-fill) ───────────────────────────────────────
    if fund_df.height:
        fund_renamed = fund_df.rename({
            "BPS": "bps", "PER": "per", "PBR": "pbr",
            "EPS": "eps", "DIV": "div", "DPS": "dps",
        })
        # asof join: 각 (code, date)에 대해 해당 날짜 이전 가장 최근 fundamental 값 사용
        # Polars에는 join_asof가 있지만, 여기서는 먼저 full join 후 sort+forward-fill 방식 사용
        fund_long = fund_renamed.sort(["code", "date"])
        # 날짜별 정렬 후 base에 nearest-backward merge
        base = base.sort(["code", "date"])
        base = base.join_asof(
            fund_long,
            on="date",
            by="code",
            strategy="backward",
        )
        # forward-fill null (앞선 날짜 데이터로 채움)
        for col in ["per", "pbr", "bps", "eps", "div", "dps"]:
            if col in base.columns:
                base = base.with_columns(
                    pl.col(col).cast(pl.Float64).fill_null(0.0)
                )
    else:
        for col in ["per", "pbr", "bps", "eps", "div", "dps"]:
            base = base.with_columns(pl.lit(0.0).alias(col))

    return base


def extended_feature_columns(df: pl.DataFrame) -> list[str]:
    """기존 피처 + 확장 피처 중 존재하는 것만 반환."""
    base_cols = feature_columns(df)
    extra = [c for c in EXTRA_FEATURE_COLS if c in df.columns]
    return base_cols + extra


# ──────────────────────────────────────────────────────────────────────────────
# 모델 학습/추론
# ──────────────────────────────────────────────────────────────────────────────

def train_lgbm(train_x: np.ndarray, train_y: np.ndarray, eval_x: np.ndarray) -> np.ndarray:
    pos = train_y.sum()
    neg = len(train_y) - pos
    scale = neg / max(pos, 1.0)
    model = lgb.LGBMClassifier(
        n_estimators=200,
        learning_rate=0.05,
        num_leaves=31,
        scale_pos_weight=scale,
        random_state=42,
        n_jobs=-1,
        verbose=-1,
    )
    model.fit(train_x, train_y.astype(int))
    return model.predict_proba(eval_x)[:, 1]


def train_xgb(train_x: np.ndarray, train_y: np.ndarray, eval_x: np.ndarray) -> np.ndarray:
    pos = train_y.sum()
    neg = len(train_y) - pos
    scale = neg / max(pos, 1.0)
    model = xgb.XGBClassifier(
        n_estimators=200,
        learning_rate=0.05,
        max_depth=6,
        scale_pos_weight=scale,
        random_state=42,
        n_jobs=-1,
        eval_metric="logloss",
        verbosity=0,
    )
    model.fit(train_x, train_y.astype(int))
    return model.predict_proba(eval_x)[:, 1]


def train_catboost(train_x: np.ndarray, train_y: np.ndarray, eval_x: np.ndarray) -> np.ndarray:
    pos = float(train_y.sum())
    neg = float(len(train_y) - pos)
    model = cb.CatBoostClassifier(
        iterations=200,
        learning_rate=0.05,
        depth=6,
        scale_pos_weight=neg / max(pos, 1.0),
        random_seed=42,
        verbose=0,
        thread_count=-1,
    )
    model.fit(train_x, train_y.astype(int))
    return model.predict_proba(eval_x)[:, 1]


# ──────────────────────────────────────────────────────────────────────────────
# 포지션 선택 및 성과 집계
# ──────────────────────────────────────────────────────────────────────────────

def select_positions(
    df: pl.DataFrame,
    prob: np.ndarray,
    pred_gap: np.ndarray,
    threshold: float,
    max_positions: int,
    min_gap: float,
) -> tuple[pl.DataFrame, dict[str, float]]:
    scored = df.with_columns([
        pl.Series("_prob", prob),
        pl.Series("_pred_gap", pred_gap),
        (
            (pl.Series("_prob2", prob) >= threshold)
            & (pl.Series("_pred_gap2", pred_gap) >= min_gap)
        ).cast(pl.Int8).alias("signal"),
    ])
    signals = scored.filter(pl.col("signal") == 1).sort(
        ["date", "_prob"], descending=[False, True]
    )
    picked = signals.group_by("date", maintain_order=True).head(max_positions) if signals.height else signals

    if picked.height == 0:
        return picked, {"selected": 0.0, "precision": 0.0, "avg_return": 0.0, "sum_return": 0.0}

    metrics = {
        "selected": float(picked.height),
        "precision": float(picked.select(pl.col("label_success").mean()).item()),
        "avg_return": float(picked.select(pl.col("overnight_return").mean()).item()),
        "sum_return": float(
            picked.group_by("date")
            .agg(pl.col("overnight_return").mean().alias("d"))
            .select(pl.col("d").sum())
            .item()
        ),
    }
    return picked, metrics


# ──────────────────────────────────────────────────────────────────────────────
# Walk-Forward
# ──────────────────────────────────────────────────────────────────────────────

MODEL_NAMES = ["mlp", "lgbm", "xgb", "catboost"]


def run_walkforward(
    candidates: pl.DataFrame,
    features: list[str],
    cfg: Config,
    start_year: int,
    end_year: int,
    threshold: float,
    max_positions: int,
    device: torch.device,
) -> dict[str, list[tuple]]:
    """연도별 walk-forward, 각 모델별 결과 리턴"""
    results: dict[str, list[tuple]] = {m: [] for m in MODEL_NAMES}

    for eval_year in range(start_year, end_year + 1):
        train_df = candidates.filter(pl.col("year") < eval_year)
        eval_df  = candidates.filter(pl.col("year") == eval_year)
        if train_df.height < 100 or eval_df.height == 0:
            print(f"  [year={eval_year}] train={train_df.height} eval={eval_df.height} → skip")
            continue

        print(f"  [year={eval_year}] train={train_df.height} eval={eval_df.height}")

        train_x_raw = train_df.select(features).to_numpy()
        eval_x_raw  = eval_df.select(features).to_numpy()
        train_x, eval_x, _ = impute_scale(train_x_raw, eval_x_raw)

        train_y_cls = train_df["label_success"].to_numpy().astype(np.float32)
        train_y_reg = train_df["next_gap_pct"].to_numpy().astype(np.float32)

        # 회귀 (공통): pred_next_gap_pct
        pred_gap = train_regressor(train_x, train_y_reg, eval_x, cfg, device)

        scored_base = eval_df.with_columns(
            pl.col("next_gap_pct").alias("overnight_return")
        )

        for model_name in MODEL_NAMES:
            try:
                if model_name == "mlp":
                    prob = train_classifier_model(
                        MLPClassifier(in_dim=train_x.shape[1], hidden_dim=cfg.hidden_dim),
                        train_x, train_y_cls, eval_x, cfg, device,
                    )
                elif model_name == "lgbm":
                    prob = train_lgbm(train_x, train_y_cls, eval_x)
                elif model_name == "xgb":
                    prob = train_xgb(train_x, train_y_cls, eval_x)
                elif model_name == "catboost":
                    prob = train_catboost(train_x, train_y_cls, eval_x)
                else:
                    continue

                _, metrics = select_positions(
                    scored_base, prob, pred_gap,
                    threshold=threshold,
                    max_positions=max_positions,
                    min_gap=cfg.min_gap,
                )
                results[model_name].append((
                    eval_year,
                    int(train_df.height),
                    int(metrics["selected"]),
                    float(metrics["precision"]),
                    float(metrics["avg_return"]),
                    float(metrics["sum_return"]),
                ))
            except Exception as e:
                print(f"  [{model_name}] year={eval_year} error: {e}")
                results[model_name].append((eval_year, int(train_df.height), 0, 0.0, 0.0, 0.0))

    return results


# ──────────────────────────────────────────────────────────────────────────────
# 차트 생성
# ──────────────────────────────────────────────────────────────────────────────

COLORS = {"mlp": "#1565c0", "lgbm": "#2e7d32", "xgb": "#e65100", "catboost": "#6a1b9a"}


def save_comparison_charts(
    results_baseline: dict[str, list[tuple]],
    results_extended: dict[str, list[tuple]],
    out_dir: Path,
) -> None:
    out_dir.mkdir(parents=True, exist_ok=True)

    # ── 연도별 sum_return 막대 비교 차트 ─────────────────────────────────────
    fig, axes = plt.subplots(1, 2, figsize=(16, 6), sharey=False)

    for ax, (results, title) in zip(
        axes,
        [(results_baseline, "Baseline (기존 피처)"), (results_extended, "Extended (확장 피처)")],
    ):
        years_all = sorted({r[0] for rows in results.values() for r in rows})
        x = np.arange(len(years_all))
        w = 0.2
        for i, m in enumerate(MODEL_NAMES):
            if not results[m]:
                continue
            row_map = {r[0]: r[5] for r in results[m]}
            vals = [row_map.get(y, 0.0) for y in years_all]
            ax.bar(x + i * w, vals, w, label=m, color=COLORS[m], alpha=0.85)
        ax.set_xticks(x + w * 1.5)
        ax.set_xticklabels(years_all)
        ax.set_title(title)
        ax.set_ylabel("Sum Return")
        ax.legend()
        ax.grid(axis="y", alpha=0.3)

    plt.tight_layout()
    plt.savefig(out_dir / "model_comparison_yearly.png", dpi=160)
    plt.close()

    # ── 전체 sum_return 막대 비교 ─────────────────────────────────────────────
    fig, ax = plt.subplots(figsize=(10, 5))
    baseline_totals = {m: sum(r[5] for r in rows) for m, rows in results_baseline.items()}
    extended_totals = {m: sum(r[5] for r in rows) for m, rows in results_extended.items()}

    x = np.arange(len(MODEL_NAMES))
    w = 0.35
    ax.bar(x - w / 2, [baseline_totals[m] for m in MODEL_NAMES], w,
           label="Baseline", color="#78909c", alpha=0.85)
    ax.bar(x + w / 2, [extended_totals[m] for m in MODEL_NAMES], w,
           label="Extended", color=[COLORS[m] for m in MODEL_NAMES], alpha=0.85)

    ax.set_xticks(x)
    ax.set_xticklabels(MODEL_NAMES)
    ax.set_title("Total Sum Return: Baseline vs Extended Features")
    ax.set_ylabel("Sum Return")
    ax.legend()
    ax.grid(axis="y", alpha=0.3)

    for i, m in enumerate(MODEL_NAMES):
        bt = baseline_totals[m]
        et = extended_totals[m]
        diff = et - bt
        ax.text(i + w / 2, et + 0.3, f"{diff:+.1f}", ha="center", fontsize=8, color="black")

    plt.tight_layout()
    plt.savefig(out_dir / "total_return_comparison.png", dpi=160)
    plt.close()


# ──────────────────────────────────────────────────────────────────────────────
# 리포트 생성
# ──────────────────────────────────────────────────────────────────────────────

def to_md_table(rows: list[tuple], headers: list[str]) -> str:
    lines = ["| " + " | ".join(headers) + " |", "|" + "|".join(["---"] * len(headers)) + "|"]
    for r in rows:
        vals = []
        for v in r:
            if isinstance(v, float):
                vals.append(f"{v:.4f}")
            else:
                vals.append(str(v))
        lines.append("| " + " | ".join(vals) + " |")
    return "\n".join(lines)


def build_report(
    results_baseline: dict[str, list[tuple]],
    results_extended: dict[str, list[tuple]],
    extended_feature_list: list[str],
    baseline_feature_list: list[str],
    out_dir: Path,
    args: argparse.Namespace,
) -> str:
    headers = ["eval_year", "train_rows", "selected", "precision", "avg_return", "sum_return"]

    sections = [f"""# Extended Feature Pipeline — 모델 비교 리포트

## 실행 설정
- DB: {args.db_path}
- 평가 기간: {args.start_year} ~ {args.end_year}
- min_gap: {args.min_gap:.3f} ({args.min_gap:.1%})
- vol_multiplier: {args.vol_multiplier}
- threshold: {args.threshold}
- max_positions/day: {args.max_positions}

## 피처 구성

### 기존(Baseline) 피처 ({len(baseline_feature_list)}개)
```
{', '.join(baseline_feature_list)}
```

### 확장(Extended) 피처 ({len(extended_feature_list)}개)
```
{', '.join(extended_feature_list)}
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

"""]

    # 전체 요약 테이블
    summary_rows = []
    for m in MODEL_NAMES:
        bt = results_baseline[m]
        et = results_extended[m]
        b_sum = sum(r[5] for r in bt)
        e_sum = sum(r[5] for r in et)
        b_prec = np.mean([r[3] for r in bt]) if bt else 0.0
        e_prec = np.mean([r[3] for r in et]) if et else 0.0
        b_sel = sum(r[2] for r in bt)
        e_sel = sum(r[2] for r in et)
        summary_rows.append([
            m,
            b_sel, f"{b_prec:.4f}", f"{b_sum:.2f}",
            e_sel, f"{e_prec:.4f}", f"{e_sum:.2f}",
            f"{e_sum - b_sum:+.2f}",
        ])

    sections.append("## 전체 성과 요약 (2021–2026)\n")
    sections.append("| model | base_selected | base_prec | base_sum | ext_selected | ext_prec | ext_sum | delta_sum |")
    sections.append("|---|---|---|---|---|---|---|---|")
    for r in summary_rows:
        sections.append("| " + " | ".join(str(x) for x in r) + " |")
    sections.append("")

    # 모델별 연도별 상세
    sections.append("\n## 모델별 walk-forward 상세\n")
    for m in MODEL_NAMES:
        sections.append(f"### {m.upper()}\n")
        sections.append("**Baseline (기존 피처)**\n")
        sections.append(to_md_table(results_baseline[m], headers))
        sections.append("\n**Extended (확장 피처)**\n")
        sections.append(to_md_table(results_extended[m], headers))
        b_total = sum(r[5] for r in results_baseline[m])
        e_total = sum(r[5] for r in results_extended[m])
        sections.append(f"\n> Baseline sum_return: **{b_total:.4f}** | Extended sum_return: **{e_total:.4f}** | Delta: **{e_total-b_total:+.4f}**\n")

    return "\n".join(sections)


# ──────────────────────────────────────────────────────────────────────────────
# CLI
# ──────────────────────────────────────────────────────────────────────────────

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser()
    p.add_argument("--db-path", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/candle_data.db"))
    p.add_argument("--meta-db", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/meta.db"))
    p.add_argument("--ta-db", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/trade_amount.db"))
    p.add_argument("--fund-db", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/fundamental.db"))
    p.add_argument("--start-year", type=int, default=2021)
    p.add_argument("--end-year", type=int, default=2026)
    p.add_argument("--min-gap", type=float, default=0.005)
    p.add_argument("--vol-multiplier", type=float, default=3.0)
    p.add_argument("--limitup-exclude-rate", type=float, default=1.0)
    p.add_argument("--threshold", type=float, default=0.50)
    p.add_argument("--max-positions", type=int, default=2)
    p.add_argument("--report-path", type=Path, default=Path("docs/extended_model_comparison.md"))
    return p.parse_args()


def make_cfg(args: argparse.Namespace) -> Config:
    return Config(
        db_path=args.db_path,
        metadata_csv=None,
        train_start="2015-01-01",
        train_end=f"{args.end_year}-12-31",
        test_start=f"{args.start_year}-01-01",
        test_end=f"{args.end_year}-12-31",
        optimize_start=f"{args.start_year}-01-01",
        optimize_end=f"{args.end_year}-12-31",
        min_gap=args.min_gap,
        vol_multiplier=args.vol_multiplier,
        limitup_exclude_rate=args.limitup_exclude_rate,
        batch_size=512,
        epochs=20,
        lr=1e-3,
        weight_decay=1e-5,
        hidden_dim=128,
        image_window=30,
        image_size=64,
        threshold_min=0.50,
        threshold_max=0.85,
        threshold_step=0.05,
        max_positions_grid=[1, 2, 3, 5, 8, 10],
        min_opt_trades=30,
        trade_penalty=2.0,
        disable_optimization=True,
        fixed_threshold=args.threshold,
        fixed_max_positions=args.max_positions,
        max_symbols=None,
        report_path=args.report_path,
    )


def main() -> None:
    args = parse_args()
    cfg = make_cfg(args)
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"Device: {device}")

    # ── 데이터 로드 ──────────────────────────────────────────────────────────
    print("Loading candle data...")
    conn = sqlite3.connect(cfg.db_path)
    symbols = list_symbol_tables(conn)
    raw = load_ohlcv(conn, symbols)
    conn.close()

    print("Loading meta / trade_amount / fundamental...")
    meta_df = load_meta(args.meta_db)

    ta_conn = sqlite3.connect(args.ta_db)
    ta_syms = [r[0] for r in ta_conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table'"
    ).fetchall()]
    ta_conn.close()
    print(f"  Loading trade_amount ({len(ta_syms)} symbols, since {args.start_year})...")
    ta_df = load_trade_amount(args.ta_db, ta_syms, since_year=2015)
    print(f"  trade_amount rows: {ta_df.height}")

    print(f"  Loading fundamental (since 2015)...")
    fund_df = load_fundamental(args.fund_db, since_year=2015)
    print(f"  fundamental rows: {fund_df.height}")

    # ── 피처 생성 (베이스라인) ────────────────────────────────────────────────
    print("Building baseline features...")
    base_data = build_features(raw, cfg, metadata=None)
    base_sup = base_data.drop_nulls(["next_gap_pct", "label_success"])
    base_cands = base_sup.filter(pl.col("is_gap_volume_candidate")).with_columns(
        pl.col("date").dt.year().alias("year")
    )
    base_feats = feature_columns(base_cands)
    print(f"  Baseline candidates: {base_cands.height}, features: {len(base_feats)}")

    # ── 피처 생성 (확장) ──────────────────────────────────────────────────────
    print("Building extended features...")
    ext_data = build_extended_features(raw, cfg, meta_df, ta_df, fund_df)
    ext_sup = ext_data.drop_nulls(["next_gap_pct", "label_success"])
    ext_cands = ext_sup.filter(pl.col("is_gap_volume_candidate")).with_columns(
        pl.col("date").dt.year().alias("year")
    )
    ext_feats = extended_feature_columns(ext_cands)
    print(f"  Extended candidates: {ext_cands.height}, features: {len(ext_feats)}")

    # ── Walk-Forward (Baseline) ───────────────────────────────────────────────
    print("\n=== Baseline Walk-Forward ===")
    results_baseline = run_walkforward(
        base_cands, base_feats, cfg,
        args.start_year, args.end_year,
        args.threshold, args.max_positions, device,
    )

    # ── Walk-Forward (Extended) ───────────────────────────────────────────────
    print("\n=== Extended Walk-Forward ===")
    results_extended = run_walkforward(
        ext_cands, ext_feats, cfg,
        args.start_year, args.end_year,
        args.threshold, args.max_positions, device,
    )

    # ── 차트 및 리포트 ────────────────────────────────────────────────────────
    out_dir = args.report_path.parent / "outputs" / "extended"
    out_dir.mkdir(parents=True, exist_ok=True)
    save_comparison_charts(results_baseline, results_extended, out_dir)

    report = build_report(
        results_baseline, results_extended,
        ext_feats, base_feats,
        out_dir, args,
    )
    args.report_path.parent.mkdir(parents=True, exist_ok=True)
    args.report_path.write_text(report, encoding="utf-8")

    # 결과 요약 print
    print("\n=== 결과 요약 ===")
    print(f"{'Model':<12} {'Base Sum':>10} {'Ext Sum':>10} {'Delta':>8}")
    print("-" * 44)
    for m in MODEL_NAMES:
        b = sum(r[5] for r in results_baseline[m])
        e = sum(r[5] for r in results_extended[m])
        print(f"{m:<12} {b:>10.2f} {e:>10.2f} {e-b:>+8.2f}")
    print(f"\nReport → {args.report_path}")
    print(f"Charts → {out_dir}")


if __name__ == "__main__":
    main()
