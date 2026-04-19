import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[3]
if str(ROOT_DIR) not in sys.path:
    sys.path.insert(0, str(ROOT_DIR))

from shared.benchmark.plotting import generate_run_graphs, generate_suite_graphs


if __name__ == "__main__":
    results_dir = Path(__file__).resolve().parent / 'results'
    if results_dir.exists():
        for run in results_dir.iterdir():
            if run.is_dir():
                generate_run_graphs(run)
        generate_suite_graphs(results_dir)
