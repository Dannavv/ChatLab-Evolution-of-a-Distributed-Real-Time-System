import json
import os
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent
RESULTS_DIR = BASE_DIR / "results"
RAW_RESULTS_DIR = BASE_DIR / "benchmark" / "results" / "raw"
LABS_DIR = BASE_DIR / "labs"
OUTPUT_FILE = RESULTS_DIR / "comparison.md"


def get_available_labs():
    if not LABS_DIR.exists():
        return []
    return sorted([d.name for d in LABS_DIR.iterdir() if d.is_dir()])


def metric_value(metrics, key, stat):
    metric = metrics.get(key, {})
    values = metric.get("values", {})
    if stat in values:
        return values[stat]
    if stat in metric:
        return metric[stat]
    return 0.0


def parse_k6_summary(path):
    try:
        with path.open("r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception:
        return None

    metrics = data.get("metrics", {})
    p50 = metric_value(metrics, "chat_message_latency_ms", "p(50)")
    p95 = metric_value(metrics, "chat_message_latency_ms", "p(95)")
    p99 = metric_value(metrics, "chat_message_latency_ms", "p(99)")
    if not p95:
        p50 = metric_value(metrics, "ws_connecting", "p(50)")
        p95 = metric_value(metrics, "ws_connecting", "p(95)")
        p99 = metric_value(metrics, "ws_connecting", "p(99)")

    throughput = metric_value(metrics, "chat_messages_sent", "rate")
    if not throughput:
        throughput = metric_value(metrics, "messages_sent", "rate")

    dropped = metric_value(metrics, "chat_messages_dropped", "count")
    if not dropped:
        dropped = metric_value(metrics, "chat_dropped_messages_total", "value")

    sent_count = metric_value(metrics, "chat_messages_sent", "count")
    if not sent_count:
        sent_count = metric_value(metrics, "messages_sent", "count")

    total = float((sent_count or 0.0) + (dropped or 0.0))
    drop_rate_pct = (float(dropped or 0.0) / total * 100.0) if total > 0 else 0.0
    tail_amp = float((p99 or 0.0) / (p95 or 1.0)) if (p95 or 0.0) > 0 else 0.0

    if (p99 or 0.0) <= 3000 and drop_rate_pct <= 1.0 and tail_amp <= 1.7:
        hypothesis = "H1_OK"
    elif (p99 or 0.0) <= 5000 and drop_rate_pct <= 3.0 and tail_amp <= 2.0:
        hypothesis = "H1_WARN"
    else:
        hypothesis = "H1_FAIL"

    return {
        "p50_latency": float(p50 or 0.0),
        "p95_latency": float(p95 or 0.0),
        "p99_latency": float(p99 or 0.0),
        "throughput": float(throughput or 0.0),
        "drop_rate_pct": float(drop_rate_pct),
        "tail_amp": float(tail_amp),
        "hypothesis": hypothesis,
    }


def load_latest_raw_run_by_lab():
    results = {}
    if not RAW_RESULTS_DIR.exists():
        return results

    for run_dir in sorted(RAW_RESULTS_DIR.iterdir()):
        if not run_dir.is_dir():
            continue

        metadata_path = run_dir / "metadata.json"
        summary_path = run_dir / "k6_summary.json"
        if not metadata_path.exists() or not summary_path.exists():
            continue

        try:
            metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
        except Exception:
            continue

        lab = metadata.get("lab")
        started_at = metadata.get("started_at_utc", "")
        if not lab:
            continue

        parsed = parse_k6_summary(summary_path)
        if not parsed:
            continue

        current = results.get(lab)
        if current is None or started_at > current["started_at_utc"]:
            results[lab] = {
                "started_at_utc": started_at,
                "workload": metadata.get("workload", "unknown"),
                **parsed,
            }

    return results


def parse_legacy_k6_results(lab_name):
    file_path = RESULTS_DIR / f"{lab_name}_k6.json"
    if not file_path.exists():
        return None

    parsed = parse_k6_summary(file_path)
    if not parsed:
        return None

    return {
        "started_at_utc": "",
        "workload": "legacy",
        **parsed,
    }


def generate_report():
    labs = get_available_labs()
    if not labs:
        print("No labs found in labs/ directory.")
        return

    raw_runs = load_latest_raw_run_by_lab()

    report = "# Benchmark Comparison Report\n\n"
    report += "| Lab | Workload | P50 (ms) | P95 (ms) | P99 (ms) | Tail Amp (p99/p95) | Throughput (msgs/sec) | Drop Rate (%) | Hypothesis | Status |\n"
    report += "|---|---|---|---|---|---|---|---|---|---|\n"

    for lab in labs:
        stats = raw_runs.get(lab) or parse_legacy_k6_results(lab)
        if stats:
            status = "PASS" if stats["hypothesis"] != "H1_FAIL" else "FAIL"
            report += (
                f"| {lab} | {stats['workload']} | {stats['p50_latency']:.2f} | {stats['p95_latency']:.2f} | "
                f"{stats['p99_latency']:.2f} | {stats['tail_amp']:.2f} | {stats['throughput']:.2f} | "
                f"{stats['drop_rate_pct']:.2f} | {stats['hypothesis']} | {status} |\n"
            )
        else:
            report += f"| {lab} | - | - | - | - | - | - | - | - | NO DATA |\n"

    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    OUTPUT_FILE.write_text(report, encoding="utf-8")
    print(f"Report generated at {OUTPUT_FILE}")


if __name__ == "__main__":
    generate_report()
