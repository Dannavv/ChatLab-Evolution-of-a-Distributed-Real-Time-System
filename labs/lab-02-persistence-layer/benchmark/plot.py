import json
import os
from pathlib import Path
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np

# GitHub Modern Style Configuration (Retina Specification)
plt.style.use('default')
plt.rcParams.update({
    "font.family": "sans-serif",
    "font.sans-serif": ["Inter", "Helvetica", "Arial", "DejaVu Sans"],
    "font.size": 10,
    "lines.linewidth": 3,
    "figure.dpi": 500,
    "axes.grid": True,
    "grid.alpha": 0.3,
    "grid.linestyle": "--",
    "axes.edgecolor": "#d0d7de",
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.2,
})

def _prepare_frame(csv_path):
    if not Path(csv_path).exists():
        return None
    try:
        df = pd.read_csv(csv_path)
        if 'timestamp_s' not in df.columns:
            df = df.rename(columns={'timestamp': 'timestamp_s'})
        df = df[df['timestamp_s'] >= 3].copy()
        df['latency_smooth'] = df['latency_ms'].rolling(window=15, min_periods=1, center=True).mean()
        if 'db_latency_ms' in df.columns:
            df['db_latency_smooth'] = df['db_latency_ms'].rolling(window=15, min_periods=1, center=True).mean()
        return df
    except Exception:
        return None

def plot_latency_scaling(df, path):
    fig, ax = plt.subplots(figsize=(8, 4))
    ax.plot(df['timestamp_s'], df['latency_ms'], color='#cf222e', alpha=0.05)
    ax.plot(df['timestamp_s'], df['latency_smooth'], color='#cf222e', label='E2E Latency')
    if 'db_latency_smooth' in df.columns:
        ax.plot(df['timestamp_s'], df['db_latency_smooth'], color='#0969da', linestyle='--', label='SQL Latency')
    ax.set_yscale('log')
    ax.set_ylabel('ms')
    ax.set_title('Performance Scaling Profile', fontweight='bold')
    ax.legend()
    _save(fig, path)

def plot_reliability_loss(df, path):
    fig, ax = plt.subplots(figsize=(8, 4))
    # Heuristic for expected rate based on VUs
    df['expected_rate'] = df['vus'] / 5.0
    msgs_diff = df['messages_total'].diff().fillna(0)
    df['actual_rate'] = msgs_diff.rolling(window=30, min_periods=1).mean()
    deficit = (df['expected_rate'] - df['actual_rate']).clip(lower=0)
    ax.fill_between(df['timestamp_s'], deficit, color='#cf222e', alpha=0.1)
    ax.plot(df['timestamp_s'], deficit, color='#cf222e')
    ax.set_ylabel('Msgs/sec Lost')
    ax.set_title('Throughput Deficit (Reliability Loss)', fontweight='bold')
    _save(fig, path)

def plot_quad_dashboard(df, path, scenario):
    fig, axes = plt.subplots(2, 2, figsize=(12, 9))
    axes[0,0].plot(df['timestamp_s'], df['latency_smooth'], color='#cf222e')
    axes[0,0].set_yscale('log')
    axes[0,0].set_title('Latency (ms)')
    
    axes[0,1].fill_between(df['timestamp_s'], df['vus'], color='#0969da', alpha=0.1)
    axes[0,1].plot(df['timestamp_s'], df['vus'], color='#0969da')
    axes[0,1].set_title('Workload (VUs)')
    
    if 'db_latency_smooth' in df.columns:
        axes[1,0].plot(df['timestamp_s'], df['db_latency_smooth'], color='#0969da')
    else:
        axes[1,0].text(0.5, 0.5, 'N/A (Memory Only)', ha='center', va='center')
    axes[1,0].set_title('SQL Overhead (ms)')
    
    axes[1,1].plot(df['timestamp_s'], df['memory_mb'], color='#8250df')
    axes[1,1].set_title('Memory (MB)')
    
    plt.suptitle(f'Audit Dashboard: {scenario.upper()}', fontsize=14, fontweight='bold')
    plt.tight_layout(rect=[0, 0.03, 1, 0.95])
    _save(fig, path)

def _save(fig, path):
    path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(path, facecolor='white', transparent=False)
    plt.close(fig)

def generate_run_graphs(run_dir):
    run_dir = Path(run_dir)
    df = _prepare_frame(run_dir / 'timeseries.csv')
    if df is None: return
    
    graphs_dir = run_dir / 'graphs'
    scenario = run_dir.name.split('__')[1] if '__' in run_dir.name else run_dir.name
    
    plot_latency_scaling(df, graphs_dir / 'modern_latency_scaling.png')
    plot_reliability_loss(df, graphs_dir / 'modern_reliability_loss.png')
    plot_quad_dashboard(df, graphs_dir / 'modern_quad_dashboard.png', scenario)

def generate_suite_graphs(results_root):
    results_root = Path(results_root)
    # Find latest lab-specific run
    runs = sorted([d for d in results_root.iterdir() if d.is_dir()], key=lambda x: x.stat().st_mtime, reverse=True)
    if not runs: return
    
    latest_run = runs[0]
    df = _prepare_frame(latest_run / 'timeseries.csv')
    if df is None: return

    assets_dir = results_root.parent.parent / 'assets' / 'benchmarks'
    scenario = latest_run.name.split('__')[1] if '__' in latest_run.name else latest_run.name

    plot_latency_scaling(df, assets_dir / 'modern_latency_scaling.png')
    plot_reliability_loss(df, assets_dir / 'modern_reliability_loss.png')
    plot_quad_dashboard(df, assets_dir / 'modern_quad_dashboard.png', scenario)
    print(f"✅ Suite assets exported to: {assets_dir}")

if __name__ == "__main__":
    results_dir = Path(__file__).resolve().parent / 'results'
    if results_dir.exists():
        for run in results_dir.iterdir():
            if run.is_dir():
                generate_run_graphs(run)
        generate_suite_graphs(results_dir)