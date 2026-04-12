#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sqlite3
import sys
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np
import polars as pl
import torch

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


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Tabular-only walk-forward overnight backtest")
    parser.add_argument("--db-path", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/candle_data.db"))
    parser.add_argument("--start-year", type=int, default=2021)
    parser.add_argument("--end-year", type=int, default=2026)
    parser.add_argument("--min-gap", type=float, default=0.01)
    parser.add_argument("--vol-multiplier", type=float, default=3.0)
    parser.add_argument("--limitup-exclude-rate", type=float, default=0.295)
    parser.add_argument("--threshold", type=float, default=0.50)
    parser.add_argument("--max-positions", type=int, default=2)
    parser.add_argument("--report-path", type=Path, default=Path("docs/tabular_walkforward_2021_2026.md"))
    return parser.parse_args()


def build_cfg(args: argparse.Namespace) -> Config:
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


def select_positions(
    df: pl.DataFrame,
    threshold: float,
    max_positions: int,
    min_gap: float,
) -> tuple[pl.DataFrame, pl.DataFrame, dict[str, float]]:
    scored = df.with_columns(
        ((pl.col("prob_tab") >= threshold) & (pl.col("pred_next_gap_pct") >= min_gap)).cast(pl.Int8).alias("signal")
    )
    signals = scored.filter(pl.col("signal") == 1).sort(["date", "prob_tab"], descending=[False, True])
    picked = signals.group_by("date", maintain_order=True).head(max_positions) if signals.height else signals

    if picked.height == 0:
        daily = pl.DataFrame({"date": [], "trades": [], "avg_return": [], "sum_return": [], "precision_day": []})
        metrics = {"selected": 0.0, "precision": 0.0, "avg_return": 0.0, "sum_return": 0.0}
        return picked, daily, metrics

    daily = (
        picked.group_by("date")
        .agg(
            [
                pl.len().alias("trades"),
                pl.col("overnight_return").mean().alias("avg_return"),
                pl.col("overnight_return").sum().alias("sum_return"),
                pl.col("label_success").mean().alias("precision_day"),
            ]
        )
        .sort("date")
    )
    metrics = {
        "selected": float(picked.height),
        "precision": float(picked.select(pl.col("label_success").mean()).item()),
        "avg_return": float(picked.select(pl.col("overnight_return").mean()).item()),
        "sum_return": float(picked.select(pl.col("overnight_return").sum()).item()),
    }
    return picked, daily, metrics


def to_markdown_table(df: pl.DataFrame, max_rows: int = 40) -> str:
    if df.height == 0:
        return "(empty)"
    head = df.head(max_rows)
    cols = head.columns
    lines = ["| " + " | ".join(cols) + " |", "|" + "|".join(["---"] * len(cols)) + "|"]
    for row in head.iter_rows(named=True):
        vals = []
        for col in cols:
            value = row[col]
            if isinstance(value, float):
                vals.append(f"{value:.6f}")
            else:
                vals.append(str(value))
        lines.append("| " + " | ".join(vals) + " |")
    return "\n".join(lines)


def save_charts(daily: pl.DataFrame, monthly: pl.DataFrame, out_dir: Path) -> tuple[Path, Path]:
    out_dir.mkdir(parents=True, exist_ok=True)
    equity_path = out_dir / "tabular_equity_curve_2021_2026.png"
    monthly_path = out_dir / "tabular_monthly_returns_2021_2026.png"

    if daily.height:
        dates = daily["date"].to_list()
        daily_avg = daily["avg_return"].to_numpy()
        equity = np.cumsum(daily_avg)
    else:
        dates = []
        equity = np.array([])

    plt.figure(figsize=(12, 5))
    if len(dates):
        plt.plot(dates, equity, linewidth=2, color="#1565c0")
        plt.axhline(0, color="black", linewidth=0.8, linestyle="--")
    plt.title("Tabular Walk-Forward Equity Curve (2021–2026)")
    plt.xlabel("Date")
    plt.ylabel("Cumulative Avg Return (sum over days)")
    plt.grid(alpha=0.3)
    plt.tight_layout()
    plt.savefig(equity_path, dpi=160)
    plt.close()

    plt.figure(figsize=(12, 5))
    if monthly.height:
        periods = [str(x) for x in monthly["period"].to_list()]
        values = monthly["sum_return"].to_numpy()
        colors = ["#2e7d32" if v >= 0 else "#c62828" for v in values]
        plt.bar(periods, values, color=colors)
        plt.xticks(rotation=60, ha="right")
    plt.title("Tabular Monthly Sum Return (2021-2026)")
    plt.xlabel("Month")
    plt.ylabel("Monthly sum_return")
    plt.grid(axis="y", alpha=0.3)
    plt.tight_layout()
    plt.savefig(monthly_path, dpi=160)
    plt.close()

    return equity_path, monthly_path


def main() -> None:
    args = parse_args()
    cfg = build_cfg(args)

    conn = sqlite3.connect(cfg.db_path)
    raw = load_ohlcv(conn, list_symbol_tables(conn))
    data = build_features(raw, cfg, None)
    supervised = data.drop_nulls(["next_gap_pct", "label_success"])
    candidates = supervised.filter(pl.col("is_gap_volume_candidate")).with_columns(pl.col("date").dt.year().alias("year"))

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    features = feature_columns(candidates)

    yearly_metrics: list[tuple[int, int, int, float, float, float]] = []
    selected_frames: list[pl.DataFrame] = []
    daily_frames: list[pl.DataFrame] = []

    for eval_year in range(args.start_year, args.end_year + 1):
        train_df = candidates.filter(pl.col("year") < eval_year)
        eval_df = candidates.filter(pl.col("year") == eval_year)
        if train_df.height == 0 or eval_df.height == 0:
            continue

        train_x_raw = train_df.select(features).to_numpy()
        eval_x_raw = eval_df.select(features).to_numpy()
        train_x, eval_x, _ = impute_scale(train_x_raw, eval_x_raw)

        train_y_reg = train_df["next_gap_pct"].to_numpy().astype(np.float32)
        train_y_cls = train_df["label_success"].to_numpy().astype(np.float32)

        pred_gap = train_regressor(train_x, train_y_reg, eval_x, cfg, device)
        prob_tab = train_classifier_model(MLPClassifier(in_dim=train_x.shape[1], hidden_dim=cfg.hidden_dim), train_x, train_y_cls, eval_x, cfg, device)

        scored = eval_df.with_columns(
            [
                pl.Series("pred_next_gap_pct", pred_gap),
                pl.Series("prob_tab", prob_tab),
                pl.col("next_gap_pct").alias("overnight_return"),
            ]
        )

        picked, daily, metrics = select_positions(scored, args.threshold, args.max_positions, args.min_gap)
        if picked.height:
            selected_frames.append(picked)
        if daily.height:
            daily_frames.append(daily)
        yearly_metrics.append(
            (
                eval_year,
                int(train_df.height),
                int(metrics["selected"]),
                float(metrics["precision"]),
                float(metrics["avg_return"]),
                float(metrics["sum_return"]),
            )
        )

    all_selected = pl.concat(selected_frames, how="vertical_relaxed") if selected_frames else pl.DataFrame()
    all_daily = pl.concat(daily_frames, how="vertical_relaxed") if daily_frames else pl.DataFrame({"date": [], "trades": [], "avg_return": [], "sum_return": [], "precision_day": []})
    all_daily = all_daily.sort("date") if all_daily.height else all_daily
    weekly = aggregate_period(all_daily, "1w")
    monthly = aggregate_period(all_daily, "1mo")
    yearly = pl.DataFrame(
        yearly_metrics,
        schema=["eval_year", "train_rows", "selected", "precision", "avg_return", "sum_return"],
        orient="row",
    )

    out_dir = args.report_path.parent / "outputs" / "tabular_walkforward"
    out_dir.mkdir(parents=True, exist_ok=True)
    if all_selected.height:
        all_selected.write_csv(out_dir / "selected_positions_2021_2026.csv")
    all_daily.write_csv(out_dir / "daily_returns_2021_2026.csv")
    weekly.write_csv(out_dir / "weekly_returns_2021_2026.csv")
    monthly.write_csv(out_dir / "monthly_returns_2021_2026.csv")
    yearly.write_csv(out_dir / "yearly_summary_2021_2026.csv")

    equity_path, monthly_path = save_charts(all_daily, monthly, out_dir)

    total_selected = int(all_selected.height) if all_selected.height else 0
    total_precision = float(all_selected.select(pl.col("label_success").mean()).item()) if all_selected.height else 0.0
    total_avg_return = float(all_selected.select(pl.col("overnight_return").mean()).item()) if all_selected.height else 0.0
    total_sum_return = float(all_selected.select(pl.col("overnight_return").sum()).item()) if all_selected.height else 0.0

    report = f"""# Tabular Walk-Forward Backtest (2021-2026)

## 설정
- 모델: tabular only
- 평가 방식: expanding walk-forward
- 평가 연도: {args.start_year} ~ {args.end_year}
- 최소 gap: {args.min_gap:.2%}
- 거래량 배수: 전일 거래량 >= 전일 60일 평균 x {args.vol_multiplier}
- 상한가/근접 제외: {args.limitup_exclude_rate:.2%}
- 고정 threshold: {args.threshold:.2f}
- 고정 max_positions/day: {args.max_positions}
- 사용 디바이스: {device.type}

## 전체 성과
- 전체 선택 종목 수: {total_selected}
- 전체 precision: {total_precision:.4f}
- 전체 avg_return: {total_avg_return:.6f}
- 전체 sum_return: {total_sum_return:.6f}

## 연도별 요약
{to_markdown_table(yearly, max_rows=20)}

## 일별 수익률
{to_markdown_table(all_daily, max_rows=120)}

## 주별 수익률
{to_markdown_table(weekly, max_rows=120)}

## 월별 수익률
{to_markdown_table(monthly, max_rows=120)}

## 차트
- Equity Curve: {equity_path}
- Monthly Return Chart: {monthly_path}
"""

    args.report_path.write_text(report, encoding="utf-8")
    print(f"Report saved: {args.report_path}")
    print(f"Output dir: {out_dir}")


if __name__ == "__main__":
    main()