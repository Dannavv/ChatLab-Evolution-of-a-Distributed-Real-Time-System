import json
import os
from pathlib import Path
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import pandas as pd
import numpy as np

# GitHub Modern Style Configuration
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
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.2,
    "axes.grid": True,
    "grid.alpha": 0.3,
    "grid.linestyle": "--",
    "axes.edgecolor": "#d0d7de",
    "axes.labelcolor": "#24292f",
    "xtick.color": "#57606a",
    "ytick.color": "#57606a",
})

def _prepare_frame(csv_path):
    df = pd.read_csv(csv_path)
    if 'timestamp' in df.columns and 'timestamp_s' not in df.columns:
        df = df.rename(columns={'timestamp': 'timestamp_s'})
    
    # Filter initial Docker noise
    df = df[df['timestamp_s'] >= 3].copy()
    
    # Clean up latency (floor at 0.1ms for clean log rendering)
    df.loc[df['latency_ms'] <= 0, 'latency_ms'] = 0.1
    
    # Cap extreme latency spikes (outliers) to keep linear graphs readable
    q95 = df['latency_ms'].quantile(0.95)
    if q95 > 0:
        df['latency_ms'] = df['latency_ms'].clip(upper=q95 * 3)
        
    # Rolling averages for "Liquid" trendlines
    df['latency_smooth'] = df['latency_ms'].rolling(window=8, min_periods=1, center=True).mean()
    df['vus_smooth'] = df['vus'].rolling(window=4, min_periods=1).mean()
    
    return df

def plot_latency_vs_vus(df, path):
    """Modern GitHub-style scatter with smooth trendline."""
    fig, ax = plt.subplots(figsize=(8, 5))
    
    # Raw samples (Subtle background)
    ax.scatter(df['vus'], df['latency_ms'], color='#57606a', s=12, alpha=0.15, label='Raw latency')
    
    # Median Trend (GitHub Blue)
    bins = np.linspace(df['vus'].min(), df['vus'].max(), 25)
    df['vu_bin'] = pd.cut(df['vus'], bins=bins)
    binned = df.groupby('vu_bin')['latency_ms'].median()
    bin_centers = [b.mid for b in binned.index]
    
    ax.plot(bin_centers, binned.values, color='#0969da', marker='o', markersize=4, 
            linewidth=2.5, label='Median performance')
    
    ax.set_xlabel('Virtual Users (Concurrency)')
    ax.set_ylabel('Latency (ms)')
    ax.set_title('Performance Scaling Profile', fontweight='bold', pad=15)
    ax.legend(frameon=True, facecolor='white', framealpha=1)
    
    # Clean the border
    for spine in ax.spines.values():
        spine.set_color('#d0d7de')
        
    _save(fig, path)

def plot_dropped_messages(df, path):
    """Clean modern step-area chart for drops."""
    fig, ax = plt.subplots(figsize=(8, 4))
    
    # Area fill (GitHub Red)
    ax.fill_between(df['timestamp_s'], df['dropped_total'], step="post", color='#cf222e', alpha=0.08)
    ax.step(df['timestamp_s'], df['dropped_total'], where='post', color='#cf222e', linewidth=2, label='Dropped messages')
    
    ax.set_xlabel('Timeline (seconds)')
    ax.set_ylabel('Cumulative Drops')
    ax.set_title('Reliability Decay', fontweight='bold', pad=15)
    ax.legend()
    
    for spine in ax.spines.values():
        spine.set_color('#d0d7de')
        
    _save(fig, path)

def plot_overview_mesh(df, path, scenario):
    """Unified multi-panel dashboard for READMEs."""
    fig, axes = plt.subplots(2, 2, figsize=(12, 9))
    
    # 1. Latency (The "Pain" metric)
    ax = axes[0,0]
    ax.plot(df['timestamp_s'], df['latency_smooth'], color='#cf222e', linewidth=2)
    ax.set_ylabel('Latency (ms)')
    ax.set_title('Latency over Time', fontweight='bold')
    
    # 2. VUs (The "Load" metric)
    ax = axes[0,1]
    ax.fill_between(df['timestamp_s'], df['vus'], color='#0969da', alpha=0.1)
    ax.plot(df['timestamp_s'], df['vus'], color='#0969da', linewidth=2)
    ax.set_ylabel('Virtual Users')
    ax.set_title('Traffic Profile', fontweight='bold')
    
    # 3. Throughput (The "Utility" metric)
    ax = axes[1,0]
    msgs_diff = df['messages_total'].diff().fillna(0)
    time_diff = df['timestamp_s'].diff().fillna(1)
    throughput = (msgs_diff / time_diff).rolling(8).mean()
    ax.plot(df['timestamp_s'], throughput, color='#1a7f37', linewidth=2)
    ax.set_ylabel('Msgs/sec')
    ax.set_title('System Throughput', fontweight='bold')
    
    # 4. Memory (The "Cost" metric)
    ax = axes[1,1]
    ax.plot(df['timestamp_s'], df['memory_mb'], color='#8250df', linewidth=2)
    ax.set_ylabel('Memory (MB)')
    ax.set_title('Resource Utilization', fontweight='bold')
    
    for a in axes.flat:
        a.set_xlabel('Seconds')
        for spine in a.spines.values():
            spine.set_color('#d0d7de')
    
    plt.suptitle(f'Benchmark Insights: {scenario.upper()}', fontsize=14, fontweight='bold', y=0.98)
    plt.tight_layout(rect=[0, 0.03, 1, 0.95])
    _save(fig, path)

def _save(fig, path):
    path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(path, facecolor='white', edgecolor='none', transparent=False)
    plt.close(fig)

def generate_run_graphs(run_dir):
    run_dir = Path(run_dir)
    csv_path = run_dir / 'timeseries.csv'
    graphs_dir = run_dir / 'graphs'
    
    if not csv_path.exists():
        return
        
    df = _prepare_frame(csv_path)
    if df.empty:
        return

    scenario = run_dir.name.split('__')[1] if '__' in run_dir.name else run_dir.name
    
    # Modern GitHub Suite
    plot_latency_vs_vus(df, graphs_dir / 'modern_latency_scaling.png')
    plot_dropped_messages(df, graphs_dir / 'modern_reliability_loss.png')
    plot_overview_mesh(df, graphs_dir / 'modern_quad_dashboard.png', scenario)

def generate_suite_graphs(results_root):
    """Generate comparison graphs across all scenarios in the suite."""
    results_root = Path(results_root)
    suite_dir = results_root / 'suite'
    suite_dir.mkdir(parents=True, exist_ok=True)
    
    scenario_runs = {}
    for run_dir in results_root.iterdir():
        if not run_dir.is_dir() or run_dir.name == 'suite':
            continue
        meta_path = run_dir / 'metadata.json'
        if not meta_path.exists():
            continue
        try:
            with open(meta_path) as f:
                meta = json.load(f)
            scenario = meta.get('scenario', 'unknown')
            started_at = meta.get('started_at_utc', '')
            
            # Keep the latest run for each scenario
            if scenario not in scenario_runs or started_at > scenario_runs[scenario]['started_at']:
                csv_path = run_dir / 'timeseries.csv'
                if csv_path.exists():
                    scenario_runs[scenario] = {'started_at': started_at, 'csv': csv_path}
        except Exception:
            continue

    if not scenario_runs:
        return

    # Prepare data for plotting comparisons
    frames = {scen: _prepare_frame(data['csv']) for scen, data in scenario_runs.items()}
    
    # 1. Latency Comparison
    fig, ax = plt.subplots(figsize=(8, 5))
    for scen, df in frames.items():
        ax.plot(df['vus'], df['latency_smooth'], label=scen, linewidth=2)
    ax.set_xlabel('Virtual Users (Concurrency)')
    ax.set_ylabel('Latency (ms)')
    ax.set_title('Lab 01 Suite: Latency Comparison', fontweight='bold', pad=15)
    ax.legend(frameon=True, facecolor='white', framealpha=1)
    for spine in ax.spines.values():
        spine.set_color('#d0d7de')
    _save(fig, suite_dir / 'suite_latency_comparison.png')

    # 2. Throughput Comparison
    fig, ax = plt.subplots(figsize=(8, 5))
    for scen, df in frames.items():
        msgs_diff = df['messages_total'].diff().fillna(0)
        time_diff = df['timestamp_s'].diff().fillna(1)
        throughput = (msgs_diff / time_diff).rolling(window=5, min_periods=1).mean()
        ax.plot(df['vus'], throughput, label=scen, linewidth=2)
    ax.set_xlabel('Virtual Users (Concurrency)')
    ax.set_ylabel('Messages / Second')
    ax.set_title('Lab 01 Suite: Throughput Comparison', fontweight='bold', pad=15)
    ax.legend(frameon=True, facecolor='white', framealpha=1)
    for spine in ax.spines.values():
        spine.set_color('#d0d7de')
    _save(fig, suite_dir / 'suite_throughput_comparison.png')

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        target = sys.argv[1]
        print(f"🚀 Processing: {target}")
        generate_run_graphs(target)
    else:
        script_dir = Path(__file__).resolve().parent
        results_dir = script_dir / 'results'
        if results_dir.exists():
            runs = [d for d in results_dir.iterdir() if d.is_dir()]
            for run in sorted(runs):
                print(f"📊 Creating Modern Dashboard for: {run.name}")
                generate_run_graphs(run)
        else:
            print("❌ No results found.")
