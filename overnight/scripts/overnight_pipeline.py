#!/usr/bin/env python3
from __future__ import annotations

import argparse
import math
import sqlite3
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import numpy as np
import polars as pl
import torch
from sklearn.metrics import classification_report, precision_score, roc_auc_score
from sklearn.preprocessing import StandardScaler
from torch import nn
from torch.utils.data import DataLoader, TensorDataset


@dataclass
class Config:
    db_path: Path
    metadata_csv: Path | None
    train_start: str
    train_end: str
    test_start: str
    test_end: str
    optimize_start: str
    optimize_end: str
    min_gap: float
    vol_multiplier: float
    limitup_exclude_rate: float
    batch_size: int
    epochs: int
    lr: float
    weight_decay: float
    hidden_dim: int
    image_window: int
    image_size: int
    threshold_min: float
    threshold_max: float
    threshold_step: float
    max_positions_grid: list[int]
    min_opt_trades: int
    trade_penalty: float
    disable_optimization: bool
    fixed_threshold: float
    fixed_max_positions: int
    max_symbols: int | None
    report_path: Path


class MLPRegressor(nn.Module):
    def __init__(self, in_dim: int, hidden_dim: int = 128) -> None:
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(in_dim, hidden_dim),
            nn.ReLU(),
            nn.Dropout(0.2),
            nn.Linear(hidden_dim, hidden_dim // 2),
            nn.ReLU(),
            nn.Dropout(0.1),
            nn.Linear(hidden_dim // 2, 1),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.net(x).squeeze(-1)


class MLPClassifier(nn.Module):
    def __init__(self, in_dim: int, hidden_dim: int = 128) -> None:
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(in_dim, hidden_dim),
            nn.ReLU(),
            nn.Dropout(0.2),
            nn.Linear(hidden_dim, hidden_dim // 2),
            nn.ReLU(),
            nn.Dropout(0.1),
            nn.Linear(hidden_dim // 2, 1),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.net(x).squeeze(-1)


class CandleCNN(nn.Module):
    def __init__(self, image_size: int) -> None:
        super().__init__()
        self.conv = nn.Sequential(
            nn.Conv2d(1, 16, 3, padding=1),
            nn.ReLU(),
            nn.MaxPool2d(2),
            nn.Conv2d(16, 32, 3, padding=1),
            nn.ReLU(),
            nn.MaxPool2d(2),
            nn.Conv2d(32, 64, 3, padding=1),
            nn.ReLU(),
            nn.AdaptiveAvgPool2d((4, 4)),
        )
        self.head = nn.Sequential(
            nn.Flatten(),
            nn.Linear(64 * 4 * 4, 64),
            nn.ReLU(),
            nn.Dropout(0.1),
            nn.Linear(64, 1),
        )
        self.image_size = image_size

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        feat = self.conv(x)
        return self.head(feat).squeeze(-1)


def parse_args() -> Config:
    parser = argparse.ArgumentParser(description="Overnight strategy pipeline (precision/returns focused)")
    parser.add_argument("--db-path", type=Path, default=Path("/mnt/data/projects/kiwoom_restapi/backend/sqlite3/candle_data.db"))
    parser.add_argument("--metadata-csv", type=Path, default=None)
    parser.add_argument("--train-start", type=str, default="2021-01-01")
    parser.add_argument("--train-end", type=str, default="2025-12-31")
    parser.add_argument("--test-start", type=str, default="2026-01-01")
    parser.add_argument("--test-end", type=str, default="2026-12-31")
    parser.add_argument("--optimize-start", type=str, default="2025-01-01")
    parser.add_argument("--optimize-end", type=str, default="2025-12-31")
    parser.add_argument("--min-gap", type=float, default=0.02)
    parser.add_argument("--vol-multiplier", type=float, default=6.0)
    parser.add_argument("--limitup-exclude-rate", type=float, default=0.295)
    parser.add_argument("--batch-size", type=int, default=512)
    parser.add_argument("--epochs", type=int, default=20)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--weight-decay", type=float, default=1e-5)
    parser.add_argument("--hidden-dim", type=int, default=128)
    parser.add_argument("--image-window", type=int, default=30)
    parser.add_argument("--image-size", type=int, default=64)
    parser.add_argument("--threshold-min", type=float, default=0.50)
    parser.add_argument("--threshold-max", type=float, default=0.85)
    parser.add_argument("--threshold-step", type=float, default=0.05)
    parser.add_argument("--max-positions-grid", type=str, default="1,2,3,5,8,10")
    parser.add_argument("--min-opt-trades", type=int, default=30)
    parser.add_argument("--trade-penalty", type=float, default=2.0)
    parser.add_argument("--disable-optimization", action="store_true")
    parser.add_argument("--fixed-threshold", type=float, default=0.50)
    parser.add_argument("--fixed-max-positions", type=int, default=1)
    parser.add_argument("--max-symbols", type=int, default=None)
    parser.add_argument("--report-path", type=Path, default=Path("docs/overnight_ml_report.md"))
    args = parser.parse_args()

    max_positions_grid = [int(x.strip()) for x in args.max_positions_grid.split(",") if x.strip()]
    if not max_positions_grid:
        max_positions_grid = [1, 2, 3, 5]

    return Config(
        db_path=args.db_path,
        metadata_csv=args.metadata_csv,
        train_start=args.train_start,
        train_end=args.train_end,
        test_start=args.test_start,
        test_end=args.test_end,
        optimize_start=args.optimize_start,
        optimize_end=args.optimize_end,
        min_gap=args.min_gap,
        vol_multiplier=args.vol_multiplier,
        limitup_exclude_rate=args.limitup_exclude_rate,
        batch_size=args.batch_size,
        epochs=args.epochs,
        lr=args.lr,
        weight_decay=args.weight_decay,
        hidden_dim=args.hidden_dim,
        image_window=args.image_window,
        image_size=args.image_size,
        threshold_min=args.threshold_min,
        threshold_max=args.threshold_max,
        threshold_step=args.threshold_step,
        max_positions_grid=max_positions_grid,
        min_opt_trades=args.min_opt_trades,
        trade_penalty=args.trade_penalty,
        disable_optimization=args.disable_optimization,
        fixed_threshold=args.fixed_threshold,
        fixed_max_positions=args.fixed_max_positions,
        max_symbols=args.max_symbols,
        report_path=args.report_path,
    )


def list_symbol_tables(conn: sqlite3.Connection, max_symbols: int | None = None) -> list[str]:
    rows = conn.execute(
        """
        SELECT name
        FROM sqlite_master
        WHERE type='table' AND name NOT LIKE 'sqlite_%'
        ORDER BY name
        """
    ).fetchall()
    symbols = [r[0] for r in rows]
    if max_symbols is not None:
        symbols = symbols[:max_symbols]
    return symbols


def quote_identifier(name: str) -> str:
    return '"' + name.replace('"', '""') + '"'


def load_ohlcv(conn: sqlite3.Connection, symbols: Iterable[str]) -> pl.DataFrame:
    frames: list[pl.DataFrame] = []
    for symbol in symbols:
        query = f"""
        SELECT '{symbol}' AS symbol, date, open, high, low, close, volume
        FROM {quote_identifier(symbol)}
        """
        try:
            df = pl.read_database(query=query, connection=conn)
        except Exception:
            continue
        if df.height:
            frames.append(df)

    if not frames:
        raise RuntimeError("No OHLCV loaded from DB")

    return pl.concat(frames, how="vertical_relaxed")


def load_metadata(path: Path | None) -> pl.DataFrame | None:
    if path is None or not path.exists():
        return None
    meta = pl.read_csv(path)
    required = {"symbol", "market_cap"}
    if not required.issubset(set(meta.columns)):
        raise ValueError("metadata csv must include: symbol, market_cap")
    return meta.select([c for c in meta.columns if c in {"symbol", "market_cap", "market", "name", "sector"}])


def build_features(df: pl.DataFrame, cfg: Config, metadata: pl.DataFrame | None) -> pl.DataFrame:
    d = (
        df.with_columns(
            [
                pl.col("date").str.strptime(pl.Date, strict=False),
                pl.col("open").cast(pl.Float64),
                pl.col("high").cast(pl.Float64),
                pl.col("low").cast(pl.Float64),
                pl.col("close").cast(pl.Float64),
                pl.col("volume").cast(pl.Float64),
            ]
        )
        .drop_nulls(["date", "open", "high", "low", "close", "volume"])
        .sort(["symbol", "date"])
        .with_row_index("row_id")
    )

    d = d.with_columns(
        [
            pl.col("close").shift(1).over("symbol").alias("prev_close"),
            pl.col("volume").shift(1).over("symbol").alias("prev_volume"),
            pl.col("volume").rolling_mean(60).over("symbol").alias("vol_ma60"),
            pl.col("volume").shift(1).rolling_mean(60).over("symbol").alias("prev_vol_ma60"),
            pl.col("close").rolling_mean(5).over("symbol").alias("ma5"),
            pl.col("close").rolling_mean(50).over("symbol").alias("ma50"),
            pl.col("close").rolling_mean(100).over("symbol").alias("ma100"),
            pl.col("close").rolling_mean(200).over("symbol").alias("ma200"),
            pl.col("close").rolling_mean(400).over("symbol").alias("ma400"),
            pl.col("close").pct_change().over("symbol").alias("ret_1d"),
            pl.col("close").pct_change(5).over("symbol").alias("ret_5d"),
            pl.col("close").pct_change(20).over("symbol").alias("ret_20d"),
            ((pl.col("high") - pl.col("low")) / pl.col("open")).alias("intraday_range"),
            ((pl.col("close") - pl.col("open")) / pl.col("open")).alias("intraday_return"),
            pl.col("open").shift(-1).over("symbol").alias("next_open"),
        ]
    )

    d = d.with_columns(
        [
            ((pl.col("open") / pl.col("prev_close")) - 1.0).alias("gap_pct"),
            ((pl.col("close") / pl.col("prev_close")) - 1.0).alias("close_vs_prev_close_ret"),
            ((pl.col("next_open") / pl.col("close")) - 1.0).alias("next_gap_pct"),
            ((pl.col("close") / pl.col("ma5")) - 1.0).alias("dist_ma5"),
            ((pl.col("close") / pl.col("ma50")) - 1.0).alias("dist_ma50"),
            ((pl.col("close") / pl.col("ma100")) - 1.0).alias("dist_ma100"),
            ((pl.col("close") / pl.col("ma200")) - 1.0).alias("dist_ma200"),
            ((pl.col("close") / pl.col("ma400")) - 1.0).alias("dist_ma400"),
        ]
    )

    d = d.with_columns(pl.col("close").diff().over("symbol").alias("delta"))
    d = d.with_columns(
        [
            pl.when(pl.col("delta") > 0).then(pl.col("delta")).otherwise(0.0).alias("gain"),
            pl.when(pl.col("delta") < 0).then(-pl.col("delta")).otherwise(0.0).alias("loss"),
        ]
    )
    d = d.with_columns(
        [
            pl.col("gain").rolling_mean(14).over("symbol").alias("avg_gain"),
            pl.col("loss").rolling_mean(14).over("symbol").alias("avg_loss"),
        ]
    )

    d = d.with_columns(
        [
            (
                100.0
                - (100.0 / (1.0 + (pl.col("avg_gain") / (pl.col("avg_loss") + pl.lit(1e-9)))))
            ).alias("rsi14"),
            (pl.col("close_vs_prev_close_ret") >= cfg.limitup_exclude_rate).alias("is_limitup_or_near"),
            (
                (pl.col("gap_pct") >= cfg.min_gap)
                & (pl.col("prev_volume") >= pl.col("prev_vol_ma60") * cfg.vol_multiplier)
                & (pl.col("close_vs_prev_close_ret") < cfg.limitup_exclude_rate)
            ).alias("is_gap_volume_candidate"),
            (pl.col("next_gap_pct") >= cfg.min_gap).cast(pl.Int8).alias("label_success"),
        ]
    )

    if metadata is not None:
        d = d.join(metadata, on="symbol", how="left")
    if "market_cap" not in d.columns:
        d = d.with_columns(pl.lit(None, dtype=pl.Float64).alias("market_cap"))

    return d.drop(["delta", "gain", "loss", "avg_gain", "avg_loss"])


def split_time(df: pl.DataFrame, start: str, end: str) -> pl.DataFrame:
    s = pl.lit(start).str.strptime(pl.Date, strict=False)
    e = pl.lit(end).str.strptime(pl.Date, strict=False)
    return df.filter((pl.col("date") >= s) & (pl.col("date") <= e))


def feature_columns(df: pl.DataFrame) -> list[str]:
    cols = [
        "volume",
        "prev_volume",
        "vol_ma60",
        "prev_vol_ma60",
        "ma5",
        "ma50",
        "ma100",
        "ma200",
        "ma400",
        "dist_ma5",
        "dist_ma50",
        "dist_ma100",
        "dist_ma200",
        "dist_ma400",
        "rsi14",
        "ret_1d",
        "ret_5d",
        "ret_20d",
        "intraday_range",
        "intraday_return",
        "market_cap",
    ]
    return [c for c in cols if c in df.columns]


def impute_scale(train_x: np.ndarray, eval_x: np.ndarray) -> tuple[np.ndarray, np.ndarray, StandardScaler]:
    tr = np.nan_to_num(train_x, nan=0.0, posinf=0.0, neginf=0.0)
    ev = np.nan_to_num(eval_x, nan=0.0, posinf=0.0, neginf=0.0)
    scaler = StandardScaler()
    tr = scaler.fit_transform(tr)
    ev = scaler.transform(ev)
    return tr, ev, scaler


def train_regressor(train_x: np.ndarray, train_y: np.ndarray, eval_x: np.ndarray, cfg: Config, device: torch.device) -> np.ndarray:
    model = MLPRegressor(train_x.shape[1], cfg.hidden_dim).to(device)
    opt = torch.optim.AdamW(model.parameters(), lr=cfg.lr, weight_decay=cfg.weight_decay)
    crit = nn.HuberLoss()

    ds = TensorDataset(torch.tensor(train_x, dtype=torch.float32), torch.tensor(train_y, dtype=torch.float32))
    dl = DataLoader(ds, batch_size=cfg.batch_size, shuffle=True)

    model.train()
    for _ in range(cfg.epochs):
        for xb, yb in dl:
            xb = xb.to(device)
            yb = yb.to(device)
            opt.zero_grad()
            pred = model(xb)
            loss = crit(pred, yb)
            loss.backward()
            opt.step()

    model.eval()
    with torch.no_grad():
        out = model(torch.tensor(eval_x, dtype=torch.float32, device=device)).cpu().numpy()
    return out


def train_classifier_model(
    model: nn.Module,
    train_x: np.ndarray,
    train_y: np.ndarray,
    eval_x: np.ndarray,
    cfg: Config,
    device: torch.device,
    is_image: bool = False,
) -> np.ndarray:
    pos = float(train_y.sum())
    neg = float(len(train_y) - pos)
    pos_weight = torch.tensor([neg / max(pos, 1.0)], dtype=torch.float32, device=device)
    crit = nn.BCEWithLogitsLoss(pos_weight=pos_weight)
    opt = torch.optim.AdamW(model.parameters(), lr=cfg.lr, weight_decay=cfg.weight_decay)

    x_dtype = torch.float32
    tx = torch.tensor(train_x, dtype=x_dtype)
    ty = torch.tensor(train_y, dtype=torch.float32)
    ds = TensorDataset(tx, ty)
    dl = DataLoader(ds, batch_size=cfg.batch_size if not is_image else min(cfg.batch_size, 128), shuffle=True)

    model = model.to(device)
    model.train()
    for _ in range(cfg.epochs):
        for xb, yb in dl:
            xb = xb.to(device)
            yb = yb.to(device)
            opt.zero_grad()
            logits = model(xb)
            loss = crit(logits, yb)
            loss.backward()
            opt.step()

    model.eval()
    with torch.no_grad():
        logits = model(torch.tensor(eval_x, dtype=x_dtype, device=device)).cpu().numpy()
    prob = 1.0 / (1.0 + np.exp(-logits))
    return prob


def group_symbol_arrays(all_df: pl.DataFrame) -> dict[str, dict[str, np.ndarray | dict]]:
    out: dict[str, dict[str, np.ndarray | dict]] = {}
    symbols = all_df.select("symbol").unique().to_series().to_list()
    for sym in symbols:
        sdf = all_df.filter(pl.col("symbol") == sym).sort("date")
        dates = sdf["date"].to_list()
        out[sym] = {
            "dates": np.array(dates, dtype=object),
            "open": sdf["open"].to_numpy(),
            "high": sdf["high"].to_numpy(),
            "low": sdf["low"].to_numpy(),
            "close": sdf["close"].to_numpy(),
            "volume": sdf["volume"].to_numpy(),
            "idx_map": {d: i for i, d in enumerate(dates)},
        }
    return out


def render_candle_image(op: np.ndarray, hi: np.ndarray, lo: np.ndarray, cl: np.ndarray, vol: np.ndarray, size: int) -> np.ndarray:
    img = np.zeros((size, size), dtype=np.float32)

    p_min = float(np.min(lo))
    p_max = float(np.max(hi))
    p_rng = max(p_max - p_min, 1e-9)

    w = len(op)
    for i in range(w):
        x_center = int((i + 0.5) * size / w)
        x_center = min(max(x_center, 0), size - 1)

        y_high = size - 1 - int((hi[i] - p_min) / p_rng * (size - 1) * 0.75)
        y_low = size - 1 - int((lo[i] - p_min) / p_rng * (size - 1) * 0.75)
        y_open = size - 1 - int((op[i] - p_min) / p_rng * (size - 1) * 0.75)
        y_close = size - 1 - int((cl[i] - p_min) / p_rng * (size - 1) * 0.75)

        y0, y1 = sorted((max(0, y_high), min(size - 1, y_low)))
        img[y0 : y1 + 1, x_center] = 0.6

        b0, b1 = sorted((max(0, y_open), min(size - 1, y_close)))
        x0 = max(0, x_center - 1)
        x1 = min(size - 1, x_center + 1)
        body_val = 1.0 if cl[i] >= op[i] else 0.35
        img[b0 : b1 + 1, x0 : x1 + 1] = body_val

    v = vol.astype(np.float64)
    v = v / max(float(np.max(v)), 1e-9)
    base_top = int(size * 0.8)
    for i in range(w):
        x_center = int((i + 0.5) * size / w)
        x_center = min(max(x_center, 0), size - 1)
        h = int(v[i] * (size - base_top - 1))
        if h > 0:
            img[size - h : size, x_center] = 0.5

    return img


def build_image_tensor(samples: pl.DataFrame, grouped: dict[str, dict[str, np.ndarray | dict]], window: int, size: int) -> np.ndarray:
    rows = samples.select(["symbol", "date"]).iter_rows(named=True)
    imgs: list[np.ndarray] = []
    for r in rows:
        sym = r["symbol"]
        dt = r["date"]
        if sym not in grouped:
            imgs.append(np.zeros((1, size, size), dtype=np.float32))
            continue
        g = grouped[sym]
        idx_map = g["idx_map"]
        if dt not in idx_map:
            imgs.append(np.zeros((1, size, size), dtype=np.float32))
            continue
        end = idx_map[dt]
        start = max(0, end - window + 1)
        op = g["open"][start : end + 1]
        hi = g["high"][start : end + 1]
        lo = g["low"][start : end + 1]
        cl = g["close"][start : end + 1]
        vol = g["volume"][start : end + 1]

        if len(op) < 2:
            imgs.append(np.zeros((1, size, size), dtype=np.float32))
            continue

        img = render_candle_image(op, hi, lo, cl, vol, size)
        imgs.append(img[None, :, :])

    return np.stack(imgs).astype(np.float32)


def export_model_onnx_fp16(model: nn.Module, sample_shape: tuple[int, ...], out_path: Path, input_name: str = "input") -> None:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    model = model.eval().cpu().half()
    dummy = torch.randn(*sample_shape, dtype=torch.float16)
    torch.onnx.export(
        model,
        dummy,
        str(out_path),
        input_names=[input_name],
        output_names=["logits"],
        dynamic_axes={input_name: {0: "batch"}, "logits": {0: "batch"}},
        opset_version=17,
    )


def simulate_strategy(
    df: pl.DataFrame,
    prob_col: str,
    gap_col: str,
    threshold: float,
    max_positions: int,
    min_gap: float,
) -> tuple[pl.DataFrame, dict[str, float]]:
    d = df.with_columns(
        ((pl.col(prob_col) >= threshold) & (pl.col(gap_col) >= min_gap)).cast(pl.Int8).alias("signal")
    )
    signals = d.filter(pl.col("signal") == 1).sort(["date", prob_col], descending=[False, True])

    # keep top-N per day
    if signals.height:
        picked = signals.group_by("date", maintain_order=True).head(max_positions)
    else:
        picked = signals

    if picked.height == 0:
        empty_daily = pl.DataFrame({"date": [], "trades": [], "avg_return": [], "sum_return": []})
        metrics = {"selected": 0.0, "precision": 0.0, "avg_return": 0.0, "sum_return": 0.0}
        return empty_daily, metrics

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
    return daily, metrics


def aggregate_period(daily_df: pl.DataFrame, every: str) -> pl.DataFrame:
    if daily_df.height == 0:
        return pl.DataFrame({"period": [], "trades": [], "avg_return": [], "sum_return": []})
    return (
        daily_df.with_columns(pl.col("date").dt.truncate(every).alias("period"))
        .group_by("period")
        .agg(
            [
                pl.col("trades").sum().alias("trades"),
                pl.col("avg_return").mean().alias("avg_return"),
                pl.col("sum_return").sum().alias("sum_return"),
            ]
        )
        .sort("period")
    )


def grid_optimize(df: pl.DataFrame, prob_col: str, gap_col: str, cfg: Config) -> dict[str, float]:
    best = {"threshold": 0.6, "max_positions": 3.0, "objective": -1e9}
    threshold_values = np.arange(cfg.threshold_min, cfg.threshold_max + 1e-9, cfg.threshold_step)
    for th in threshold_values:
        for mp in cfg.max_positions_grid:
            _, m = simulate_strategy(df, prob_col, gap_col, float(th), int(mp), cfg.min_gap)
            # precision first, then returns
            trade_shortfall = max(0.0, float(cfg.min_opt_trades) - m["selected"])
            penalty = cfg.trade_penalty * trade_shortfall
            objective = m["precision"] * 100.0 + m["sum_return"] - penalty
            if objective > best["objective"]:
                best = {"threshold": float(th), "max_positions": float(mp), "objective": float(objective)}
    return best


def walk_forward_summary(
    train_scored: pl.DataFrame,
    prob_col: str,
    gap_col: str,
    cfg: Config,
) -> pl.DataFrame:
    if train_scored.height == 0:
        return pl.DataFrame({"eval_year": [], "best_threshold": [], "best_max_positions": [], "selected": [], "precision": [], "sum_return": []})

    with_year = train_scored.with_columns(pl.col("date").dt.year().alias("year"))
    years = with_year.select("year").unique().sort("year").to_series().to_list()
    rows: list[tuple[int, float, int, int, float, float]] = []

    for eval_year in years:
        hist = with_year.filter(pl.col("year") < eval_year)
        eval_df = with_year.filter(pl.col("year") == eval_year)
        if hist.height == 0 or eval_df.height == 0:
            continue
        best = grid_optimize(hist, prob_col, gap_col, cfg)
        _, metrics = simulate_strategy(
            eval_df,
            prob_col,
            gap_col,
            best["threshold"],
            int(best["max_positions"]),
            cfg.min_gap,
        )
        rows.append(
            (
                int(eval_year),
                float(best["threshold"]),
                int(best["max_positions"]),
                int(metrics["selected"]),
                float(metrics["precision"]),
                float(metrics["sum_return"]),
            )
        )

    return pl.DataFrame(
        rows,
        schema=["eval_year", "best_threshold", "best_max_positions", "selected", "precision", "sum_return"],
        orient="row",
    )


def to_markdown_table(df: pl.DataFrame, max_rows: int = 30) -> str:
    if df.height == 0:
        return "(empty)"
    h = df.head(max_rows)
    cols = h.columns
    lines = ["| " + " | ".join(cols) + " |", "|" + "|".join(["---"] * len(cols)) + "|"]
    for row in h.iter_rows(named=True):
        vals = []
        for c in cols:
            v = row[c]
            if isinstance(v, float):
                vals.append(f"{v:.6f}")
            else:
                vals.append(str(v))
        lines.append("| " + " | ".join(vals) + " |")
    return "\n".join(lines)


def main() -> None:
    cfg = parse_args()
    if not cfg.db_path.exists():
        raise FileNotFoundError(f"DB not found: {cfg.db_path}")

    conn = sqlite3.connect(cfg.db_path)
    symbols = list_symbol_tables(conn, cfg.max_symbols)
    raw = load_ohlcv(conn, symbols)
    meta = load_metadata(cfg.metadata_csv)
    data = build_features(raw, cfg, meta)

    train_df = split_time(data, cfg.train_start, cfg.train_end)
    test_df = split_time(data, cfg.test_start, cfg.test_end)

    supervised_train = train_df.drop_nulls(["next_gap_pct", "label_success"])
    supervised_test = test_df.drop_nulls(["next_gap_pct", "label_success"])

    excluded_limitup_train = supervised_train.filter(pl.col("is_limitup_or_near")).height
    excluded_limitup_test = supervised_test.filter(pl.col("is_limitup_or_near")).height

    # candidate set is now based on previous-day volume condition at buy-time perspective
    cand_train = supervised_train.filter(pl.col("is_gap_volume_candidate"))
    cand_test = supervised_test.filter(pl.col("is_gap_volume_candidate"))

    opt_df = split_time(cand_train, cfg.optimize_start, cfg.optimize_end)
    fit_df = cand_train.filter(pl.col("date") < pl.lit(cfg.optimize_start).str.strptime(pl.Date, strict=False))
    if fit_df.height == 0:
        fit_df = cand_train

    features = feature_columns(data)

    fit_x = fit_df.select(features).to_numpy()
    fit_y_cls = fit_df["label_success"].to_numpy().astype(np.float32)
    fit_y_reg = fit_df["next_gap_pct"].to_numpy().astype(np.float32)

    opt_x_raw = opt_df.select(features).to_numpy()
    test_x_raw = cand_test.select(features).to_numpy()
    train_cand_x_raw = cand_train.select(features).to_numpy()

    fit_x_tab, opt_x_tab, _ = impute_scale(fit_x, opt_x_raw)
    _, test_x_tab, _ = impute_scale(fit_x, test_x_raw)
    _, train_cand_x_tab, _ = impute_scale(fit_x, train_cand_x_raw)

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")

    # regression for next gap magnitude
    pred_gap_opt = train_regressor(fit_x_tab, fit_y_reg, opt_x_tab, cfg, device)
    pred_gap_test = train_regressor(fit_x_tab, fit_y_reg, test_x_tab, cfg, device)
    pred_gap_train = train_regressor(fit_x_tab, fit_y_reg, train_cand_x_tab, cfg, device)

    # tabular classifier
    tab_model = MLPClassifier(in_dim=fit_x_tab.shape[1], hidden_dim=cfg.hidden_dim)
    prob_opt_tab = train_classifier_model(tab_model, fit_x_tab, fit_y_cls, opt_x_tab, cfg, device)
    prob_test_tab = train_classifier_model(tab_model, fit_x_tab, fit_y_cls, test_x_tab, cfg, device)
    prob_train_tab = train_classifier_model(tab_model, fit_x_tab, fit_y_cls, train_cand_x_tab, cfg, device)

    # candle image classifier
    grouped = group_symbol_arrays(data.select(["symbol", "date", "open", "high", "low", "close", "volume"]))
    fit_img = build_image_tensor(fit_df, grouped, cfg.image_window, cfg.image_size)
    opt_img = build_image_tensor(opt_df, grouped, cfg.image_window, cfg.image_size)
    test_img = build_image_tensor(cand_test, grouped, cfg.image_window, cfg.image_size)
    train_img = build_image_tensor(cand_train, grouped, cfg.image_window, cfg.image_size)

    img_model = CandleCNN(cfg.image_size)
    prob_opt_img = train_classifier_model(img_model, fit_img, fit_y_cls, opt_img, cfg, device, is_image=True)
    prob_test_img = train_classifier_model(img_model, fit_img, fit_y_cls, test_img, cfg, device, is_image=True)
    prob_train_img = train_classifier_model(img_model, fit_img, fit_y_cls, train_img, cfg, device, is_image=True)

    # with-image model: simple equal-weight ensemble
    prob_opt_fusion = 0.5 * prob_opt_tab + 0.5 * prob_opt_img
    prob_test_fusion = 0.5 * prob_test_tab + 0.5 * prob_test_img
    prob_train_fusion = 0.5 * prob_train_tab + 0.5 * prob_train_img

    opt_scored = opt_df.with_columns(
        [
            pl.Series("pred_next_gap_pct", pred_gap_opt),
            pl.Series("prob_tab", prob_opt_tab),
            pl.Series("prob_img", prob_opt_img),
            pl.Series("prob_fusion", prob_opt_fusion),
            pl.col("next_gap_pct").alias("overnight_return"),
        ]
    )

    test_scored = cand_test.with_columns(
        [
            pl.Series("pred_next_gap_pct", pred_gap_test),
            pl.Series("prob_tab", prob_test_tab),
            pl.Series("prob_img", prob_test_img),
            pl.Series("prob_fusion", prob_test_fusion),
            pl.col("next_gap_pct").alias("overnight_return"),
        ]
    )

    train_scored = cand_train.with_columns(
        [
            pl.Series("pred_next_gap_pct", pred_gap_train),
            pl.Series("prob_tab", prob_train_tab),
            pl.Series("prob_img", prob_train_img),
            pl.Series("prob_fusion", prob_train_fusion),
            pl.col("next_gap_pct").alias("overnight_return"),
        ]
    )

    # parameter selection: optimized mode or fixed mode
    if cfg.disable_optimization:
        best_tab = {
            "threshold": float(cfg.fixed_threshold),
            "max_positions": float(cfg.fixed_max_positions),
            "objective": float("nan"),
        }
        best_fusion = {
            "threshold": float(cfg.fixed_threshold),
            "max_positions": float(cfg.fixed_max_positions),
            "objective": float("nan"),
        }
        wf_tab = pl.DataFrame(
            {
                "eval_year": [],
                "best_threshold": [],
                "best_max_positions": [],
                "selected": [],
                "precision": [],
                "sum_return": [],
            }
        )
        wf_fusion = wf_tab.clone()
    else:
        best_tab = grid_optimize(opt_scored, "prob_tab", "pred_next_gap_pct", cfg)
        best_fusion = grid_optimize(opt_scored, "prob_fusion", "pred_next_gap_pct", cfg)
        wf_tab = walk_forward_summary(train_scored, "prob_tab", "pred_next_gap_pct", cfg)
        wf_fusion = walk_forward_summary(train_scored, "prob_fusion", "pred_next_gap_pct", cfg)

    # evaluate on 2026 candidate set
    daily_tab, metrics_tab = simulate_strategy(
        test_scored,
        "prob_tab",
        "pred_next_gap_pct",
        best_tab["threshold"],
        int(best_tab["max_positions"]),
        cfg.min_gap,
    )
    daily_fusion, metrics_fusion = simulate_strategy(
        test_scored,
        "prob_fusion",
        "pred_next_gap_pct",
        best_fusion["threshold"],
        int(best_fusion["max_positions"]),
        cfg.min_gap,
    )

    weekly_tab = aggregate_period(daily_tab, "1w")
    monthly_tab = aggregate_period(daily_tab, "1mo")
    weekly_fusion = aggregate_period(daily_fusion, "1w")
    monthly_fusion = aggregate_period(daily_fusion, "1mo")

    y_opt = opt_df["label_success"].to_numpy().astype(np.int32) if opt_df.height else np.array([], dtype=np.int32)
    y_test = cand_test["label_success"].to_numpy().astype(np.int32)

    auc_tab = float("nan")
    auc_fusion = float("nan")
    if y_test.size > 0 and len(np.unique(y_test)) > 1:
        auc_tab = roc_auc_score(y_test, prob_test_tab)
        auc_fusion = roc_auc_score(y_test, prob_test_fusion)

    p50_tab = precision_score(y_test, (prob_test_tab >= 0.5).astype(np.int32), zero_division=0) if y_test.size else 0.0
    p50_fusion = precision_score(y_test, (prob_test_fusion >= 0.5).astype(np.int32), zero_division=0) if y_test.size else 0.0

    # export FP16 ONNX for CPU inference preparation
    model_dir = cfg.report_path.parent / "outputs" / "models"
    export_model_onnx_fp16(MLPClassifier(in_dim=fit_x_tab.shape[1], hidden_dim=cfg.hidden_dim), (1, fit_x_tab.shape[1]), model_dir / "tabular_classifier_fp16.onnx")
    export_model_onnx_fp16(CandleCNN(cfg.image_size), (1, 1, cfg.image_size, cfg.image_size), model_dir / "candle_cnn_fp16.onnx")

    best_model_name = "fusion" if metrics_fusion["sum_return"] >= metrics_tab["sum_return"] else "tabular"

    report = f"""# Overnight 전략 학습/검증 리포트 (Precision/수익률 중심)

## 실행 설정
- DB: {cfg.db_path}
- 학습 구간: {cfg.train_start} ~ {cfg.train_end}
- 검증 구간: {cfg.test_start} ~ {cfg.test_end}
- 최적화 구간(파라미터 탐색): {cfg.optimize_start} ~ {cfg.optimize_end}
- 최소 gap: {cfg.min_gap:.2%}
- 거래량 배수 조건: 전일 거래량 >= 전일 기준 60일 평균 x {cfg.vol_multiplier}
- 상한가/근접 제외: close/prev_close - 1 >= {cfg.limitup_exclude_rate:.2%} 제외
- 사용 디바이스: {device.type}
- 전체 종목 수: {len(symbols)}
- 학습 후보 샘플 수(cand_train): {cand_train.height}
- 최적화 샘플 수(opt_2025): {opt_df.height}
- 검증 후보 샘플 수(cand_test_2026): {cand_test.height}
- 상한가/근접 제외 건수(train): {excluded_limitup_train}
- 상한가/근접 제외 건수(test): {excluded_limitup_test}
- 최적화 사용 여부: {not cfg.disable_optimization}
- 고정 threshold(최적화 비활성 시): {cfg.fixed_threshold:.2f}
- 고정 max_positions(최적화 비활성 시): {cfg.fixed_max_positions}

## 매수 시점 기준 후보 규칙(교정 반영)
1. 당일 시가 gap: `open_t / close_(t-1) - 1 >= 2%`
2. 전일 거래량 조건: `volume_(t-1) >= ma60(volume)_(t-1) * {cfg.vol_multiplier}`
3. 상한가/근접 제외: `close_t / close_(t-1) - 1 < 29.5%`

## 모델 성능(정밀도 중심)
- Tabular ROC-AUC(2026 후보셋): {auc_tab:.4f}
- Tabular+Image ROC-AUC(2026 후보셋): {auc_fusion:.4f}
- Tabular Precision@0.5: {p50_tab:.4f}
- Tabular+Image Precision@0.5: {p50_fusion:.4f}

## 파라미터 설정 결과
- threshold 탐색 범위: {cfg.threshold_min:.2f} ~ {cfg.threshold_max:.2f} (step={cfg.threshold_step:.2f})
- max_positions 탐색 집합: {cfg.max_positions_grid}
- 최소 거래수 페널티 기준: min_opt_trades={cfg.min_opt_trades}, penalty_weight={cfg.trade_penalty}
- Tabular best threshold: {best_tab['threshold']:.2f}
- Tabular best max positions/day: {int(best_tab['max_positions'])}
- Fusion best threshold: {best_fusion['threshold']:.2f}
- Fusion best max positions/day: {int(best_fusion['max_positions'])}

## Walk-forward 파라미터 안정성(학습구간 내부)
### Tabular
{to_markdown_table(wf_tab, max_rows=20)}

### Fusion
{to_markdown_table(wf_fusion, max_rows=20)}

## 2026 실검증: 전략 비교
| model | selected | precision | avg_return | sum_return |
|---|---:|---:|---:|---:|
| tabular | {int(metrics_tab['selected'])} | {metrics_tab['precision']:.4f} | {metrics_tab['avg_return']:.6f} | {metrics_tab['sum_return']:.6f} |
| tabular+image(fusion) | {int(metrics_fusion['selected'])} | {metrics_fusion['precision']:.4f} | {metrics_fusion['avg_return']:.6f} | {metrics_fusion['sum_return']:.6f} |

## 일별 수익률(Top)
### Tabular
{to_markdown_table(daily_tab, max_rows=120)}

### Tabular+Image(Fusion)
{to_markdown_table(daily_fusion, max_rows=120)}

## 주별 수익률
### Tabular
{to_markdown_table(weekly_tab, max_rows=80)}

### Tabular+Image(Fusion)
{to_markdown_table(weekly_fusion, max_rows=80)}

## 월별 수익률
### Tabular
{to_markdown_table(monthly_tab, max_rows=24)}

### Tabular+Image(Fusion)
{to_markdown_table(monthly_fusion, max_rows=24)}

## 캔들 이미지 추가 효과
이미지 추가 모델은 캔들 윈도우({cfg.image_window}일)를 {cfg.image_size}x{cfg.image_size} 흑백 이미지로 변환하여 CNN 분류기를 학습했다.

효과 판단은 2026 실검증에서 아래를 비교한다.
1. precision
2. sum_return
3. avg_return

이번 결과에서 더 나은 수익률 기준 우세 모델: {best_model_name}

## ONNX FP16 CPU 추론 준비
아래 FP16 ONNX 모델이 생성되었다.
1. {model_dir / 'tabular_classifier_fp16.onnx'}
2. {model_dir / 'candle_cnn_fp16.onnx'}
"""

    cfg.report_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.report_path.write_text(report, encoding="utf-8")

    out_dir = cfg.report_path.parent / "outputs"
    out_dir.mkdir(parents=True, exist_ok=True)
    test_scored.write_csv(out_dir / "candidate_predictions_2026.csv")
    daily_tab.write_csv(out_dir / "daily_returns_2026_tabular.csv")
    daily_fusion.write_csv(out_dir / "daily_returns_2026_fusion.csv")
    weekly_tab.write_csv(out_dir / "weekly_returns_2026_tabular.csv")
    weekly_fusion.write_csv(out_dir / "weekly_returns_2026_fusion.csv")
    monthly_tab.write_csv(out_dir / "monthly_returns_2026_tabular.csv")
    monthly_fusion.write_csv(out_dir / "monthly_returns_2026_fusion.csv")
    wf_tab.write_csv(out_dir / "walk_forward_tabular.csv")
    wf_fusion.write_csv(out_dir / "walk_forward_fusion.csv")

    if y_test.size:
        cls = classification_report(y_test, (prob_test_fusion >= best_fusion["threshold"]).astype(np.int32), digits=4)
        (out_dir / "classification_report_2026_fusion.txt").write_text(cls, encoding="utf-8")

    print(f"Report saved: {cfg.report_path}")
    print(f"Output dir: {out_dir}")


if __name__ == "__main__":
    main()
