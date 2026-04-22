#!/usr/bin/env python3
"""
Regenerate all benchmark graphs from latest comparison_standard runs.
This ensures README.md files always display the latest graphs.
"""
import sys
from pathlib import Path

from shared.benchmark.plotting import generate_run_graphs, generate_suite_graphs, refresh_lab_readme_assets


ROOT_DIR = Path(__file__).resolve().parents[1]
LABS_DIR = ROOT_DIR / "labs"


def regenerate_run_graphs():
    """Regenerate per-run graphs for all labs."""
    total = 0
    skipped = 0

    for lab_dir in sorted(LABS_DIR.glob("lab-*")):
        results_root = lab_dir / "benchmark" / "results"
        if not results_root.exists():
            continue

        # Find all comparison_standard runs
        for run_dir in sorted(results_root.iterdir()):
            if not run_dir.is_dir():
                continue
            if "comparison_standard" not in run_dir.name:
                continue

            csv_path = run_dir / "timeseries.csv"
            if not csv_path.exists():
                print(f"⚠️  Skipping {run_dir.name} (missing timeseries.csv)")
                skipped += 1
                continue

            try:
                generate_run_graphs(run_dir)
                total += 1
                print(f"✅ Regenerated graphs for: {run_dir.name}")
            except Exception as e:
                print(f"❌ Failed to regenerate {run_dir.name}: {e}")
                skipped += 1

    return total, skipped


def regenerate_suite_graphs():
    """Regenerate suite-level graphs from latest comparison_standard runs."""
    total = 0
    skipped = 0

    for lab_dir in sorted(LABS_DIR.glob("lab-*")):
        results_root = lab_dir / "benchmark" / "results"
        if not results_root.exists():
            continue

        try:
            generate_suite_graphs(results_root, root_dir=ROOT_DIR)
            total += 1
        except Exception as e:
            print(f"❌ Failed to generate suite graphs for {lab_dir.name}: {e}")
            skipped += 1

    return total, skipped


def regenerate_lab_readme_assets():
    """Refresh lab-local README assets from latest comparison_standard run."""
    total = 0
    skipped = 0

    for lab_dir in sorted(LABS_DIR.glob("lab-*")):
        results_root = lab_dir / "benchmark" / "results"
        if not results_root.exists():
            continue

        assets_dir = lab_dir / "assets" / "benchmarks"
        try:
            if refresh_lab_readme_assets(results_root, assets_dir):
                total += 1
            else:
                skipped += 1
        except Exception as e:
            print(f"❌ Failed to sync README assets for {lab_dir.name}: {e}")
            skipped += 1

    return total, skipped


def main():
    print("🎯 Regenerating all benchmark graphs from comparison_standard runs...")
    print()

    print("Phase 1: Per-run graphs")
    print("-" * 50)
    run_total, run_skipped = regenerate_run_graphs()
    print(f"✅ Regenerated {run_total} per-run graph sets, skipped {run_skipped}")
    print()

    print("Phase 2: Suite-level graphs")
    print("-" * 50)
    suite_total, suite_skipped = regenerate_suite_graphs()
    print(f"✅ Regenerated {suite_total} suite graph sets, skipped {suite_skipped}")
    print()

    print("Phase 3: Lab README assets")
    print("-" * 50)
    assets_total, assets_skipped = regenerate_lab_readme_assets()
    print(f"✅ Synced {assets_total} lab README asset sets, skipped {assets_skipped}")
    print()

    # Verify root-level suite graphs exist
    suite_graphs = [
        ROOT_DIR / "assets" / "benchmarks" / "modern_latency_scaling.png",
        ROOT_DIR / "assets" / "benchmarks" / "modern_reliability_loss.png",
        ROOT_DIR / "assets" / "benchmarks" / "modern_quad_dashboard.png",
    ]
    missing = [g for g in suite_graphs if not g.exists()]

    if missing:
        print(f"⚠️  Missing suite graphs: {[g.name for g in missing]}")
        return 1
    else:
        print("✅ All root-level suite graphs present at assets/benchmarks/")
        print()
        print("📊 README.md files will now display latest benchmark graphs.")
        return 0


if __name__ == "__main__":
    sys.exit(main())
