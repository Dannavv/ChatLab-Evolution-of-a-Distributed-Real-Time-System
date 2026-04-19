import pandas as pd
import matplotlib.pyplot as plt
import os
import numpy as np

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RESULTS_DIR = os.path.join(BASE_DIR, "results")
ASSETS_DIR = os.path.join(BASE_DIR, "assets", "benchmarks")

def plot_master_report(csv_path):
    if not os.path.exists(csv_path):
        return

    lab_name = os.path.basename(csv_path).replace("_robust_report.csv", "")
    df = pd.read_csv(csv_path)

    # --- 1. ROBUST CLEANING ---
    df = df[df['vus'] >= 10] # Ignore noisy start
    if df.empty: return
    
    peak_idx = df['vus'].idxmax()
    df = df.iloc[:peak_idx + 1].copy()

    # Remove extreme outliers in latency
    q_limit = df['latency_ms'].quantile(0.99)
    if df['latency_ms'].max() > q_limit * 5:
        df = df[df['latency_ms'] <= q_limit * 2]

    # Calculate Throughput (Msgs/Sec)
    df['rate_raw'] = df['messages_total'].diff() / df['timestamp'].diff()
    df['rate_smooth'] = df['rate_raw'].rolling(window=5, min_periods=1).mean()
    
    # --- ROBUST EFFICIENCY CALCULATION ---
    # Efficiency = Throughput / VUs
    df['efficiency'] = df['rate_smooth'] / df['vus']
    
    # Use the 95th percentile of the early-phase efficiency as the 100% baseline
    # This prevents the "500% efficiency" artifact at the start
    early_phase = df.head(int(len(df) * 0.2))
    if not early_phase.empty:
        baseline = early_phase['efficiency'].quantile(0.95)
    else:
        baseline = df['efficiency'].max()
    
    if baseline <= 0: baseline = 1
    
    df['efficiency_pct'] = (df['efficiency'] / baseline) * 100
    
    # Cap at 100% to keep the graph clean and focused on the degradation
    df['efficiency_pct'] = df['efficiency_pct'].clip(upper=100)

    # --- 2. MASTER PLOT (HIGH FIDELITY) ---
    plt.style.use('seaborn-v0_8-whitegrid')
    fig, (ax1, ax2, ax3) = plt.subplots(1, 3, figsize=(26, 9), constrained_layout=True)
    
    colors = {'lat': '#dc2626', 'thr': '#7c3aed', 'eff': '#f59e0b'}

    # Panel A: Latency
    ax1.plot(df['vus'], df['latency_ms'], color=colors['lat'], linewidth=4)
    ax1.fill_between(df['vus'], df['latency_ms'], color=colors['lat'], alpha=0.1)
    ax1.set_ylabel("LATENCY (ms)", fontsize=14, fontweight='bold', labelpad=15)
    ax1.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight='bold', labelpad=10)
    ax1.set_title("ROBUST RESPONSE SPECTRUM", fontsize=16, fontweight='bold', pad=20)
    ax1.set_ylim(0, df['latency_ms'].max() * 1.15)

    # Panel B: Throughput
    ax2.plot(df['vus'], df['rate_smooth'], color=colors['thr'], linewidth=4)
    ax2.fill_between(df['vus'], df['rate_smooth'], color=colors['thr'], alpha=0.1)
    ax2.set_ylabel("AGGREGATE MSGS / SEC", fontsize=14, fontweight='bold', labelpad=15)
    ax2.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight='bold', labelpad=10)
    ax2.set_title("ROBUST THROUGHPUT CAPACITY", fontsize=16, fontweight='bold', pad=20)

    # Panel C: Efficiency (Stability Focused)
    ax3.plot(df['vus'], df['efficiency_pct'], color=colors['eff'], linewidth=4)
    ax3.fill_between(df['vus'], df['efficiency_pct'], color=colors['eff'], alpha=0.1)
    ax3.set_ylim(0, 110)
    ax3.set_ylabel("ROBUST EFFICIENCY (%)", fontsize=14, fontweight='bold', labelpad=15)
    ax3.set_xlabel("VIRTUAL USERS", fontsize=14, fontweight='bold', labelpad=10)
    ax3.set_title("CONCURRENCY OVERHEAD", fontsize=16, fontweight='bold', pad=20)
    # Add a guide line at 80% (The Efficiency Cliff)
    ax3.axhline(y=80, color='gray', linestyle='--', alpha=0.5)

    plt.suptitle(f"ROBUST PERFORMANCE MANUSCRIPT: {lab_name.upper()} SCALE ANALYSIS", 
                 fontsize=24, fontweight='bold', y=1.08)
    
    if not os.path.exists(ASSETS_DIR): os.makedirs(ASSETS_DIR)
    output_path = os.path.join(ASSETS_DIR, f"{lab_name}-performance.png")
    plt.savefig(output_path, dpi=200, bbox_inches='tight', pad_inches=0.5)
    print(f"🏆 CLEAN ROBUST REPORT GENERATED: {output_path}")

if __name__ == "__main__":
    csvs = [f for f in os.listdir(RESULTS_DIR) if f.endswith("_robust_report.csv")]
    for csv in csvs:
        plot_master_report(os.path.join(RESULTS_DIR, csv))
