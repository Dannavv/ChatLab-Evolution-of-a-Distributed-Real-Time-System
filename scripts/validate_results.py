#!/usr/bin/env python3
import json
import sys
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
RESULTS_DIR = ROOT_DIR / "results"
COMPARISON_MD = RESULTS_DIR / "comparison.md"
COMPARISON_JSON = RESULTS_DIR / "comparison.json"


def _load_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def _extract_md_lab_rows(path):
    lines = path.read_text(encoding="utf-8").splitlines()
    rows = []
    for line in lines:
        if not line.startswith("| lab-"):
            continue
        parts = [part.strip() for part in line.strip("|").split("|")]
        if len(parts) != 10:
            continue
        rows.append(
            {
                "lab": parts[0],
                "scenario": parts[1],
                "status": parts[9],
            }
        )
    return rows


def validate_numeric_ranges(row):
    issues = []
    lab = row.get("lab")
    latency = row.get("latency_ms", {})
    throughput = row.get("throughput_msgs_s", {})
    reliability = row.get("reliability", {})

    for key in ["p50", "p90", "p95", "p99"]:
        value = latency.get(key)
        if value is not None and value < 0:
            issues.append(f"{lab}: latency {key} must be >= 0")

    for key in ["avg", "peak"]:
        value = throughput.get(key)
        if value is not None and value < 0:
            issues.append(f"{lab}: throughput {key} must be >= 0")

    error_rate = reliability.get("error_rate_pct")
    delivery = reliability.get("delivery_ratio_pct")
    duplicate = reliability.get("duplicate_ratio_pct")

    for name, value in [
        ("error_rate_pct", error_rate),
        ("delivery_ratio_pct", delivery),
        ("duplicate_ratio_pct", duplicate),
    ]:
        if value is not None and (value < 0 or value > 100):
            issues.append(f"{lab}: {name} out of range [0, 100]")

    return issues


def main():
    errors = []
    warnings = []

    if not COMPARISON_JSON.exists():
        print(f"ERROR: missing {COMPARISON_JSON}")
        return 1
    if not COMPARISON_MD.exists():
        print(f"ERROR: missing {COMPARISON_MD}")
        return 1

    payload = _load_json(COMPARISON_JSON)
    json_rows = payload.get("generated_rows", [])
    md_rows = _extract_md_lab_rows(COMPARISON_MD)

    md_map = {row["lab"]: row for row in md_rows}
    json_labs = {row.get("lab") for row in json_rows}
    md_labs = set(md_map.keys())

    missing_in_md = sorted(json_labs - md_labs)
    missing_in_json = sorted(md_labs - json_labs)

    if missing_in_md:
        errors.append("labs missing in comparison.md: " + ", ".join(missing_in_md))
    if missing_in_json:
        errors.append("labs missing in comparison.json: " + ", ".join(missing_in_json))

    for row in json_rows:
        lab = row.get("lab")
        status = row.get("status")
        reliability = row.get("reliability", {})
        totals = row.get("totals", {})

        errors.extend(validate_numeric_ranges(row))

        md_row = md_map.get(lab)
        if md_row and md_row.get("scenario") != row.get("scenario"):
            errors.append(f"{lab}: scenario mismatch between markdown and json")

        if status == "OK" and totals.get("sent", 0) == 0 and totals.get("received", 0) == 0:
            warnings.append(f"{lab}: delivery counters unavailable (sent/received are zero)")

        if status == "OK" and reliability.get("error_rate_pct", 0) > 0:
            md_status = md_row.get("status", "") if md_row else ""
            if "errors" not in md_status:
                errors.append(f"{lab}: non-zero error rate not reflected in markdown status")

    summary = {
        "checked_json_rows": len(json_rows),
        "checked_md_rows": len(md_rows),
        "errors": len(errors),
        "warnings": len(warnings),
        "status": "PASS" if not errors else "FAIL",
    }
    print(json.dumps(summary, indent=2))

    for warning in warnings:
        print(f"WARNING: {warning}")
    for error in errors:
        print(f"ERROR: {error}")

    return 0 if not errors else 1


if __name__ == "__main__":
    sys.exit(main())
