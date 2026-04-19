import os
import subprocess
import sys
from pathlib import Path

from shared.benchmark.report import build_comparison_artifacts

ROOT_DIR = Path(__file__).resolve().parent
LABS_DIR = ROOT_DIR / 'labs'

def get_labs_with_benchmarks():
    labs = []
    if not LABS_DIR.exists():
        return labs
    
    # Get all lab directories that contain a benchmark/run.py
    for lab_folder in sorted(LABS_DIR.iterdir()):
        if lab_folder.is_dir() and (lab_folder / 'benchmark' / 'run.py').exists():
            labs.append(lab_folder.name)
    return labs

def run_lab_benchmark(lab_name, scenario=None):
    lab_run_script = LABS_DIR / lab_name / 'benchmark' / 'run.py'
    if not lab_run_script.exists():
        print(f"❌ No benchmark found for {lab_name}")
        return

    cmd = [sys.executable, str(lab_run_script)]
    if scenario:
        cmd.extend(['--scenario', scenario])
    else:
        cmd.append('--all')

    print(f"\n🚀 Starting benchmark for: {lab_name.upper()}")
    print("-" * 40)
    try:
        # Run and stream output to console
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError as e:
        print(f"❌ Benchmark failed for {lab_name}: {e}")
    except KeyboardInterrupt:
        print(f"\n⚠️  Benchmark interrupted by user.")

def show_menu(labs):
    print("\n" + "═"*60)
    print(" 🚀 CHATLAB: DISTRIBUTED SYSTEMS RESEARCH CENTER")
    print(" 📡 Global Orchestrator")
    print("═"*60)
    print("0) [RUN ALL LABS (Full Regression Suite)]")
    print("c) [REBUILD COMPARISON REPORT]")
    for i, lab in enumerate(labs, 1):
        # Human readable name
        display_name = lab.replace('-', ' ').title()
        print(f"{i}) {display_name}")
    print("q) Quit")
    print("-" * 60)

def main():
    while True:
        labs = get_labs_with_benchmarks()
        if not labs:
            print("❌ No labs with benchmarks found in /labs directory.")
            print("   Make sure labs have a 'benchmark/run.py' file.")
            break
            
        show_menu(labs)
        choice = input("👉 Select Lab Index: ").strip().lower()
        
        if choice == 'q':
            print("👋 Exiting Orchestrator.")
            break

        if choice == 'c':
            report_path = build_comparison_artifacts()
            print(f"✅ Comparison report rebuilt at: {report_path}")
            continue
        
        if choice == '0':
            print(f"\n☢️  CRITICAL: STARTING GLOBAL REGRESSION ({len(labs)} Labs)...")
            for lab in labs:
                run_lab_benchmark(lab)
            report_path = build_comparison_artifacts()
            print(f"📊 Comparison report updated: {report_path}")
            print("\n✅ GLOBAL REGRESSION COMPLETE.")
            break
        
        try:
            idx = int(choice) - 1
            if 0 <= idx < len(labs):
                run_lab_benchmark(labs[idx])
            else:
                print("❌ Invalid selection.")
        except ValueError:
            print("❌ Please enter a number, 'c', or 'q'.")

if __name__ == "__main__":
    main()
