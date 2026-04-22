#!/usr/bin/env python3
import json
import sys
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
LABS_DIR = ROOT_DIR / "labs"

SLOS = {
    "latency_p95_ms_max": 50.0,
    "latency_p99_ms_max": 100.0,
    "error_rate_pct_max": 0.1,
    "delivery_ratio_pct_min": 99.9,
}


def _load_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def _latest_run_for(results_root, scenario_name):
    if not results_root.exists():
        return None
    candidates = []
    for entry in results_root.iterdir():
        if not entry.is_dir():
            continue
        metadata_path = entry / "metadata.json"
        if not metadata_path.exists():
            continue
        metadata = _load_json(metadata_path)
        if metadata.get("scenario") != scenario_name:
            continue
        candidates.append(entry)
    if not candidates:
        return None
    return max(candidates, key=lambda item: item.stat().st_mtime)


def evaluate_summary(lab, summary):
    checks = []
    latency = summary.get("latency_ms", {})
    reliability = summary.get("reliability", {})
    totals = summary.get("totals", {})

    p95 = float(latency.get("p95", 0) or 0)
    p99 = float(latency.get("p99", 0) or 0)
    error_rate = float(reliability.get("error_rate_pct", 0) or 0)
    delivery = float(reliability.get("delivery_ratio_pct", 0) or 0)
    sent = int(totals.get("sent", 0) or 0)

    checks.append(("latency_p95_ms_max", p95 <= SLOS["latency_p95_ms_max"], p95))
    checks.append(("latency_p99_ms_max", p99 <= SLOS["latency_p99_ms_max"], p99))
    checks.append(("error_rate_pct_max", error_rate <= SLOS["error_rate_pct_max"], error_rate))

    if sent > 0:
        checks.append(("delivery_ratio_pct_min", delivery >= SLOS["delivery_ratio_pct_min"], delivery))
    else:
        checks.append(("delivery_ratio_pct_min", None, delivery))

    hard_fails = [name for name, passed, _ in checks if passed is False and name != "delivery_ratio_pct_min"]
    soft_fails = [name for name, passed, _ in checks if passed is False and name == "delivery_ratio_pct_min"]
    unknown = [name for name, passed, _ in checks if passed is None]

    if hard_fails:
        status = "FAIL"
    elif soft_fails or unknown:
        status = "WARN"
    else:
        status = "PASS"

    return {
        "lab": lab,
        "status": status,
        "checks": [
            {
                "name": name,
                "threshold": SLOS[name],
                "value": value,
                "result": "PASS" if passed is True else "FAIL" if passed is False else "UNKNOWN",
            }
            for name, passed, value in checks
        ],
    }


def main():
    reports = []

    for lab_dir in sorted(LABS_DIR.glob("lab-*")):
        workload_path = lab_dir / "benchmark" / "workload.yaml"
        if not workload_path.exists():
            continue

        results_root = lab_dir / "benchmark" / "results"
        latest_run = _latest_run_for(results_root, "comparison_standard")
        if latest_run is None:
            reports.append({"lab": lab_dir.name, "status": "WARN", "reason": "missing comparable run"})
            continue

        summary_path = latest_run / "benchmark_summary.json"
        if not summary_path.exists():
            reports.append({"lab": lab_dir.name, "status": "WARN", "reason": "missing benchmark summary"})
            continue

        summary = _load_json(summary_path)
        reports.append(evaluate_summary(lab_dir.name, summary))

    aggregate = {
        "labs_checked": len(reports),
        "pass": sum(1 for row in reports if row.get("status") == "PASS"),
        "warn": sum(1 for row in reports if row.get("status") == "WARN"),
        "fail": sum(1 for row in reports if row.get("status") == "FAIL"),
    }

    print(json.dumps({"aggregate": aggregate, "reports": reports}, indent=2))

    return 1 if aggregate["fail"] > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
