#!/usr/bin/env python3
import json
import sys
from pathlib import Path

import yaml

from shared.benchmark.framework import REQUIRED_WORKLOAD_KEYS


ROOT_DIR = Path(__file__).resolve().parents[1]
LABS_DIR = ROOT_DIR / "labs"


def _load_yaml(path):
    with path.open("r", encoding="utf-8") as handle:
        return yaml.safe_load(handle) or {}


def _validate_comparison_scenario(workload, path):
    errors = []
    scenarios = workload.get("scenarios", {})
    scenario = scenarios.get("comparison_standard")
    if not isinstance(scenario, dict):
        errors.append(f"{path}: missing scenarios.comparison_standard")
        return errors

    stages = scenario.get("stages", [])
    if not stages:
        errors.append(f"{path}: comparison_standard missing stages")
    if "message_interval_ms" not in scenario:
        errors.append(f"{path}: comparison_standard missing message_interval_ms")

    return errors


def validate_workload(path):
    errors = []
    workload = _load_yaml(path)
    missing = [key for key in REQUIRED_WORKLOAD_KEYS if key not in workload]
    if missing:
        errors.append(f"{path}: missing required keys: {', '.join(missing)}")

    errors.extend(_validate_comparison_scenario(workload, path))
    return errors


def main():
    workload_files = sorted(LABS_DIR.glob("lab-*/benchmark/workload.yaml"))
    all_errors = []

    for workload_file in workload_files:
        all_errors.extend(validate_workload(workload_file))

    result = {
        "checked": len(workload_files),
        "errors": len(all_errors),
        "status": "PASS" if not all_errors else "FAIL",
    }
    print(json.dumps(result, indent=2))

    if all_errors:
        for error in all_errors:
            print(f"ERROR: {error}")
        return 1

    print("All workload manifests satisfy the benchmark contract.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
