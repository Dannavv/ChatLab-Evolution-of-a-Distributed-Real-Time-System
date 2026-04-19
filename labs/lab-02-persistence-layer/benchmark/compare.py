import json
import os
from pathlib import Path
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np

# GitHub Modern Style
plt.rcParams.update({
    "font.family": "sans-serif",
    "font.sans-serif": ["Inter", "Helvetica", "Arial", "DejaVu Sans"],
    "font.size": 10,
    "lines.linewidth": 3,
    "figure.dpi": 300,
    "savefig.bbox": "tight",
})

def get_latest_run(lab_id, preferred_scenarios=None):
    results_root = Path(__file__).resolve().parent.parent.parent / lab_id / 'benchmark' / 'results'
    if not results_root.exists():
        return None
    runs = [d for d in results_root.iterdir() if d.is_dir() and (d / 'timeseries.csv').exists()]
    if not runs:
        return None
    preferred_scenarios = preferred_scenarios or []
    preferred_runs = []
    for run in runs:
        meta_path = run / 'metadata.json'
        if not meta_path.exists():
            continue
        try:
            meta = json.loads(meta_path.read_text(encoding='utf-8'))
        except Exception:
            continue
        if meta.get('scenario') in preferred_scenarios:
            preferred_runs.append(run)
    pool = preferred_runs or runs
    return sorted(pool, key=os.path.getmtime)[-1]

def process_df(path):
    df = pd.read_csv(path)
    # Smoothing for cleaner comparisons
    df['latency_smooth'] = df['latency_ms'].rolling(window=10, min_periods=1, center=True).mean()
    # Normalize throughput (messages per second)
    msgs_diff = df['messages_total'].diff().fillna(0)
    df['tput'] = msgs_diff.rolling(window=5, min_periods=1).mean()
    return df

def generate_comparison():
    preferred = ['comparison_standard']
    lab01_run = get_latest_run('lab-01-monolith-baseline', preferred)
    lab02_run = get_latest_run('lab-02-persistence-layer', preferred)
    
    if not lab01_run or not lab02_run:
        print("Error: Missing benchmark data for Lab 01 or Lab 02.")
        return

    df1 = process_df(lab01_run / 'timeseries.csv')
    df2 = process_df(lab02_run / 'timeseries.csv')

    # 1. THE GOLDEN OVERLAY: Throughput vs Latency
    fig, ax = plt.subplots(figsize=(10, 6))
    
    # Sort by throughput to create a coherent performance frontier
    df1_sorted = df1.sort_values('tput')
    df2_sorted = df2.sort_values('tput')

    ax.plot(df1_sorted['tput'], df1_sorted['latency_smooth'], label='Lab 01: In-Memory (Baseline)', color='#0969da')
    ax.plot(df2_sorted['tput'], df2_sorted['latency_smooth'], label='Lab 02: PostgreSQL (Durable)', color='#cf222e')
    
    ax.set_yscale('log')
    ax.set_xlabel('Throughput (Messages / Second)')
    ax.set_ylabel('Latency (ms, Log Scale)')
    ax.set_title('The Performance vs. Durability Trade-off', fontweight='bold', pad=20)
    ax.legend()
    ax.grid(True, alpha=0.3, linestyle='--')
    
    assets_dir = Path(__file__).resolve().parent.parent / 'assets' / 'benchmarks'
    assets_dir.mkdir(parents=True, exist_ok=True)
    fig.savefig(assets_dir / 'lab_comparison_overlay.png', facecolor='white', transparent=False)
    
    # 2. LATENCY BREAKDOWN (Lab 02 specific)
    fig, ax = plt.subplots(figsize=(10, 6))
    if 'db_latency_ms' in df2.columns or 'sql_latency_ms' in df2.columns:
        db_col = 'db_latency_ms' if 'db_latency_ms' in df2.columns else 'sql_latency_ms'
        db_smooth = df2[db_col].rolling(window=10, min_periods=1, center=True).mean()
        app_smooth = df2['latency_smooth'] - db_smooth.clip(lower=0)
        
        ax.stackplot(df2['timestamp_s'], [app_smooth, db_smooth], labels=['Processing/Network', 'SQL Persistence'], colors=['#afb8c1', '#0969da'], alpha=0.8)
        ax.set_ylabel('Latency Breakdown (ms)')
        ax.set_xlabel('Timeline (Seconds)')
        ax.set_title('Lab 02: Internal Latency Breakdown', fontweight='bold')
        ax.legend(loc='upper left')
        fig.savefig(assets_dir / 'lab02_latency_breakdown.png', facecolor='white', transparent=False)

    print(f"✅ Comparison assets generated in {assets_dir}")

if __name__ == "__main__":
    generate_comparison()
