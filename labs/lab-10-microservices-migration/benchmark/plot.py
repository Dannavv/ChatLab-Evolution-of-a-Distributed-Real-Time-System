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
    "axes.labelsize": 10,
    "axes.titlesize": 12,
    "legend.fontsize": 9,
    "xtick.labelsize": 9,
    "ytick.labelsize": 9,
    "lines.linewidth": 3,
    "figure.dpi": 500,
    "axes.grid": True,
    "grid.alpha": 0.3,
    "grid.linestyle": "--",
    "axes.edgecolor": "#d0d7de",
    "axes.labelcolor": "#24292f",
    "xtick.color": "#57606a",
    "ytick.color": "#57606a",
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.2,
})

def _prepare_frame(csv_path):
    df = pd.read_csv(csv_path)
    if 'timestamp' in df.columns and 'timestamp_s' not in df.columns:
        df = df.rename(columns={'timestamp': 'timestamp_s'})
    
    # Filter warm-up and idle
    df = df[df['timestamp_s'] >= 3].copy()
    
    # Statistical Floor for Log Scale
    df.loc[df['latency_ms'] <= 0, 'latency_ms'] = 0.1
    if 'db_latency_ms' in df.columns:
        df.loc[df['db_latency_ms'] <= 0, 'db_latency_ms'] = 0.1
    
    # Gaussian Smoothing for "Clean" lines over noisy data (Higher window for deficit)
    df['latency_smooth'] = df['latency_ms'].rolling(window=15, min_periods=1, center=True).mean()
    if 'db_latency_ms' in df.columns:
        df['db_latency_smooth'] = df['db_latency_ms'].rolling(window=15, min_periods=1, center=True).mean()
    
    return df

def generate_run_graphs(run_dir):
    """Processes a single run folder with burst-resistant logic."""
    run_dir = Path(run_dir)
    csv_path = run_dir / 'timeseries.csv'
    graphs_dir = run_dir / 'graphs'
    if not csv_path.exists(): return
    df = _prepare_frame(csv_path)
    if df.empty: return
    
    scenario = run_dir.name.split('__')[1] if '__' in run_dir.name else run_dir.name
    
    fig, axes = plt.subplots(2, 2, figsize=(12, 9))
    
    # 1. E2E Latency (Red) - With Log Scale and Outlier Protection
    ax = axes[0,0]
    ax.plot(df['timestamp_s'], df['latency_ms'], color='#cf222e', alpha=0.05) # Raw shadow
    ax.plot(df['timestamp_s'], df['latency_smooth'], color='#cf222e', label='Total Latency')
    ax.set_yscale('log')
    ax.set_ylabel('Latency (ms)')
    ax.set_title('E2E Latency Profile', fontweight='bold')
    
    # 2. VUs (Blue)
    ax = axes[0,1]
    ax.fill_between(df['timestamp_s'], df['vus'], color='#0969da', alpha=0.1)
    ax.plot(df['timestamp_s'], df['vus'], color='#0969da')
    ax.set_ylabel('Virtual Users')
    ax.set_title('Load Profile', fontweight='bold')
    
    # 3. DB Persistence Tax (Dual-Axis or Focused Zoom)
    ax = axes[1,0]
    if 'db_latency_ms' in df.columns:
        ax.plot(df['timestamp_s'], df['db_latency_smooth'], color='#0969da', label='SQL Time')
        # Limit Y-axis to 20ms to see the detail, unless it actually spikes higher
        y_max = max(20, df['db_latency_smooth'].max() * 1.2)
        ax.set_ylim(0, y_max)
    ax.set_ylabel('SQL Latency (ms)')
    ax.set_title('Database Write Overhead', fontweight='bold')
    
    # 4. Reliability Deficit (Trend-Focused)
    ax = axes[1,1]
    df['expected_rate'] = df['vus'] / 5.0
    msgs_diff = df['messages_total'].diff().fillna(0)
    # Long window (30s) to smooth out the "Backlog Bursts"
    df['actual_rate'] = msgs_diff.rolling(window=30, min_periods=1).mean()
    deficit = (df['expected_rate'] - df['actual_rate']).clip(lower=0)
    ax.fill_between(df['timestamp_s'], deficit, color='#cf222e', alpha=0.1)
    ax.plot(df['timestamp_s'], deficit, color='#cf222e', linewidth=2)
    ax.set_ylabel('Avg Msgs/sec Lost')
    ax.set_title('Throughput Reliability Trend', fontweight='bold')

    for a in axes.flat:
        a.set_xlabel('Seconds')
        for spine in a.spines.values(): spine.set_color('#d0d7de')

    plt.suptitle(f'Run Audit: {scenario.upper()}', fontsize=14, fontweight='bold', y=0.98)
    plt.tight_layout(rect=[0, 0.03, 1, 0.95])
    _save(fig, graphs_dir / 'run_audit.png')

def generate_suite_graphs(results_root):
    """Aggregates all scenarios into final README comparison assets."""
    results_root = Path(results_root)
    assets_dir = results_root.parent.parent / 'assets' / 'benchmarks'
    assets_dir.mkdir(parents=True, exist_ok=True)
    
    # Auto-discover latest run for each unique scenario
    scenarios = {}
    for run_dir in results_root.iterdir():
        if not run_dir.is_dir(): continue
        meta_path = run_dir / 'metadata.json'
        if not meta_path.exists(): continue
        meta = json.loads(meta_path.read_text())
        name = meta.get('scenario', 'unknown')
        start_time = meta.get('started_at_utc', '')
        if name not in scenarios or start_time > scenarios[name]['time']:
            scenarios[name] = {'time': start_time, 'path': run_dir / 'timeseries.csv'}

    if not scenarios: return

    # 1. SCALING COMPARISON (Latency vs VUs across scenarios)
    fig, ax = plt.subplots(figsize=(10, 6))
    colors = ['#0969da', '#cf222e', '#1a7f37', '#8250df']
    
    for i, (name, data) in enumerate(sorted(scenarios.items())):
        df = _prepare_frame(data['path'])
        # Sort by VUs to create a proper scaling line
        scaling = df.groupby('vus')['latency_smooth'].median().sort_index()
        ax.plot(scaling.index, scaling.values, label=f"Scenario: {name}", 
                color=colors[i % len(colors)], marker='o', markersize=4)

    ax.set_yscale('log')
    ax.set_xlabel('Virtual Users (Load)')
    ax.set_ylabel('Median Latency (ms)')
    ax.set_title('Lab 02 Persistence Scaling: Comparison', fontweight='bold', pad=20)
    ax.legend(frameon=True, facecolor='white', framealpha=1)
    _save(fig, assets_dir / 'suite_scaling_comparison.png')

    # 2. PERSISTENCE TAX COMPARISON (DB Latency across scenarios)
    fig, ax = plt.subplots(figsize=(10, 6))
    for i, (name, data) in enumerate(sorted(scenarios.items())):
        df = _prepare_frame(data['path'])
        if 'db_latency_smooth' in df.columns:
            db_scaling = df.groupby('vus')['db_latency_smooth'].median().sort_index()
            ax.plot(db_scaling.index, db_scaling.values, label=f"DB Tax: {name}", 
                    color=colors[i % len(colors)], linestyle='--')

    ax.set_xlabel('Virtual Users (Load)')
    ax.set_ylabel('SQL Write Latency (ms)')
    ax.set_title('PostgreSQL Performance under Load', fontweight='bold', pad=20)
    ax.legend()
    _save(fig, assets_dir / 'suite_db_performance.png')
    
    print(f"✅ Suite assets exported to: {assets_dir}")

def _save(fig, path):
    path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(path, facecolor='white', edgecolor='none', transparent=False)
    plt.close(fig)

if __name__ == "__main__":
    import sys
    if len(sys.argv) > 1:
        generate_run_graphs(sys.argv[1])
    else:
        results_dir = Path(__file__).resolve().parent / 'results'
        if results_dir.exists():
            for run in sorted([d for d in results_dir.iterdir() if d.is_dir()]):
                generate_run_graphs(run)
            generate_suite_graphs(results_dir)