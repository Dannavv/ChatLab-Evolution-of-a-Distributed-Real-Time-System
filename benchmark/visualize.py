import json
from pathlib import Path

import matplotlib.pyplot as plt
import pandas as pd

BASE_DIR = Path(__file__).resolve().parent.parent
RESULTS_DIR = BASE_DIR / "results"
RAW_RESULTS_DIR = BASE_DIR / "benchmark" / "results" / "raw"
ASSETS_DIR = BASE_DIR / "assets" / "benchmarks"


def _normalize_timeseries(df):
    if "timestamp" in df.columns and "timestamp_s" not in df.columns:
        df = df.rename(columns={"timestamp": "timestamp_s"})

    required = ["timestamp_s", "vus", "latency_ms", "memory_mb", "messages_total"]
    missing = [c for c in required if c not in df.columns]
    if missing:
        raise ValueError(f"timeseries missing columns: {', '.join(missing)}")

    return df


def _series_clean(df):
    df = df[df["vus"] >= 10].copy()
    if df.empty:
        return df

    peak_idx = df["vus"].idxmax()
    df = df.loc[:peak_idx].copy()

    q_limit = df["latency_ms"].quantile(0.99)
    if df["latency_ms"].max() > q_limit * 5:
        df = df[df["latency_ms"] <= q_limit * 2]

    # Instantaneous throughput from cumulative message counter.
    df["rate_raw"] = df["messages_total"].diff() / df["timestamp_s"].diff()
    df["rate_smooth"] = df["rate_raw"].rolling(window=5, min_periods=1).mean()
    df["efficiency"] = df["rate_smooth"] / df["vus"]

    early_phase = df.head(max(1, int(len(df) * 0.2)))
    baseline = early_phase["efficiency"].quantile(0.95) if not early_phase.empty else df["efficiency"].max()
    if baseline <= 0:
        baseline = 1

    df["efficiency_pct"] = (df["efficiency"] / baseline) * 100
    df["efficiency_pct"] = df["efficiency_pct"].clip(upper=100)
    return df


def _output_name(lab_name, workload_name):
    if workload_name and workload_name != "robust_steady":
        return f"{lab_name}-{workload_name}-performance.png"
    return f"{lab_name}-performance.png"


def plot_master_report(df, lab_name, workload_name="robust_steady"):
    df = _series_clean(df)
    if df.empty:
        return

    plt.style.use("seaborn-v0_8-whitegrid")
    fig, (ax1, ax2, ax3) = plt.subplots(1, 3, figsize=(26, 9), constrained_layout=True)

    colors = {"lat": "#dc2626", "thr": "#7c3aed", "eff": "#f59e0b"}

    ax1.plot(df["vus"], df["latency_ms"], color=colors["lat"], linewidth=4)
    ax1.fill_between(df["vus"], df["latency_ms"], color=colors["lat"], alpha=0.1)
    ax1.set_ylabel("LATENCY (ms)", fontsize=14, fontweight="bold", labelpad=15)
    ax1.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight="bold", labelpad=10)
    ax1.set_title("RESPONSE SPECTRUM", fontsize=16, fontweight="bold", pad=20)
    ax1.set_ylim(0, df["latency_ms"].max() * 1.15)

    ax2.plot(df["vus"], df["rate_smooth"], color=colors["thr"], linewidth=4)
    ax2.fill_between(df["vus"], df["rate_smooth"], color=colors["thr"], alpha=0.1)
    ax2.set_ylabel("AGGREGATE MSGS / SEC", fontsize=14, fontweight="bold", labelpad=15)
    ax2.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight="bold", labelpad=10)
    ax2.set_title("THROUGHPUT CAPACITY", fontsize=16, fontweight="bold", pad=20)

    ax3.plot(df["vus"], df["efficiency_pct"], color=colors["eff"], linewidth=4)
    ax3.fill_between(df["vus"], df["efficiency_pct"], color=colors["eff"], alpha=0.1)
    ax3.set_ylim(0, 110)
    ax3.set_ylabel("EFFICIENCY (%)", fontsize=14, fontweight="bold", labelpad=15)
    ax3.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight="bold", labelpad=10)
    ax3.set_title("CONCURRENCY OVERHEAD", fontsize=16, fontweight="bold", pad=20)
    ax3.axhline(y=80, color="gray", linestyle="--", alpha=0.5)

    plt.suptitle(
        f"PERFORMANCE MANUSCRIPT: {lab_name.upper()} [{workload_name}]",
        fontsize=22,
        fontweight="bold",
        y=1.05,
    )

    ASSETS_DIR.mkdir(parents=True, exist_ok=True)
    output_path = ASSETS_DIR / _output_name(lab_name, workload_name)
    plt.savefig(output_path, dpi=200, bbox_inches="tight", pad_inches=0.5)
    plt.close(fig)
    print(f"Generated: {output_path}")


def _iter_latest_raw_runs_by_lab():
    latest = {}
    if not RAW_RESULTS_DIR.exists():
        return latest

    for run_dir in sorted(RAW_RESULTS_DIR.iterdir()):
        if not run_dir.is_dir():
            continue

        metadata_path = run_dir / "metadata.json"
        series_path = run_dir / "timeseries.csv"
        if not metadata_path.exists() or not series_path.exists():
            continue

        try:
            metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
        except Exception:
            continue

        lab = metadata.get("lab")
        started_at = metadata.get("started_at_utc", "")
        if not lab:
            continue

        current = latest.get(lab)
        if current is None or started_at > current["started_at_utc"]:
            latest[lab] = {
                "started_at_utc": started_at,
                "workload": metadata.get("workload", "robust_steady"),
                "series_path": series_path,
            }

    return latest


def _plot_from_raw_runs():
    latest = _iter_latest_raw_runs_by_lab()
    plotted = 0

    for lab, info in sorted(latest.items()):
        try:
            df = pd.read_csv(info["series_path"])
            df = _normalize_timeseries(df)
            plot_master_report(df, lab, info["workload"])
            plotted += 1
        except Exception as exc:
            print(f"Skipped raw run for {lab}: {exc}")

    return plotted


def _plot_from_legacy_csvs():
    plotted = 0
    if not RESULTS_DIR.exists():
        return plotted

    for csv_path in sorted(RESULTS_DIR.glob("*_robust_report.csv")):
        lab_name = csv_path.name.replace("_robust_report.csv", "")
        try:
            df = pd.read_csv(csv_path)
            df = _normalize_timeseries(df)
            plot_master_report(df, lab_name, "robust_steady")
            plotted += 1
        except Exception as exc:
            print(f"Skipped legacy csv {csv_path}: {exc}")

    return plotted


def main():
    raw_count = _plot_from_raw_runs()
    if raw_count == 0:
        legacy_count = _plot_from_legacy_csvs()
        if legacy_count == 0:
            print("No benchmark series found.")


if __name__ == "__main__":
    main()
