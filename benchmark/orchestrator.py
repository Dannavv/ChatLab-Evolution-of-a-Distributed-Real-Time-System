import csv
import datetime as dt
import hashlib
import json
import os
import re
import signal
import subprocess
import sys
import threading
import time
from pathlib import Path

import requests
import yaml

BASE_DIR = Path(__file__).resolve().parent.parent
BENCHMARK_DIR = BASE_DIR / "benchmark"
RESULTS_DIR = BASE_DIR / "results"
LABS_DIR = BASE_DIR / "labs"
RAW_RESULTS_DIR = BENCHMARK_DIR / "results" / "raw"
WORKLOADS_DIR = BENCHMARK_DIR / "workloads"
SCHEMAS_DIR = BENCHMARK_DIR / "schemas"

active_lab_dir = None
k6_process = None

# Lab metrics endpoints. Values are HTTP ports exposing /metrics.
LAB_METRICS_PORTS = {
    "lab-01-monolith-baseline": [8080],
    "lab-02-persistence-layer": [8080],
    "lab-03-redis-pubsub": [8082, 8083],
    "lab-04-scalable-monolith": [8084],
    "lab-05-cloud-native-chat-infrastructure": [8085],
    "lab-06-chaos-and-resilience": [8086, 8087],
    "lab-07-real-time-presence-and-delivery": [8088, 8089],
    "lab-08-global-multi-region": [8090, 8091, 8092],
    "lab-09-message-security": [8094, 8095],
    "lab-10-microservices-migration": [8096, 8097, 8098],
}

# Lab websocket entrypoints used by k6.
LAB_WS_PORTS = {
    "lab-01-monolith-baseline": [8080],
    "lab-02-persistence-layer": [8080],
    "lab-03-redis-pubsub": [8082, 8083],
    "lab-04-scalable-monolith": [8084],
    "lab-05-cloud-native-chat-infrastructure": [8085],
    "lab-06-chaos-and-resilience": [8086],
    "lab-07-real-time-presence-and-delivery": [8088, 8089],
    "lab-08-global-multi-region": [8090, 8091],
    "lab-09-message-security": [8094, 8095],
    "lab-10-microservices-migration": [8096],
}


def cleanup(sig=None, frame=None):
    global active_lab_dir, k6_process
    if k6_process:
        k6_process.terminate()
    if active_lab_dir:
        subprocess.run(
            "docker-compose down",
            shell=True,
            cwd=active_lab_dir,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
    print("\n🛑 Benchmark run terminated.")
    sys.exit(0)


signal.signal(signal.SIGINT, cleanup)
signal.signal(signal.SIGTERM, cleanup)


def get_available_labs():
    return sorted([d for d in os.listdir(LABS_DIR) if os.path.isdir(LABS_DIR / d)])


def get_available_workloads():
    if not WORKLOADS_DIR.exists():
        return []
    return sorted([p.stem for p in WORKLOADS_DIR.glob("*.yaml")])


def load_workload(workload_name):
    workload_path = WORKLOADS_DIR / f"{workload_name}.yaml"
    if not workload_path.exists():
        raise FileNotFoundError(f"workload not found: {workload_name}")

    with workload_path.open("r", encoding="utf-8") as f:
        data = yaml.safe_load(f)

    validate_workload_schema(data, workload_name)

    return data


def _schema_load():
    schema_path = SCHEMAS_DIR / "workload.schema.yaml"
    if not schema_path.exists():
        raise FileNotFoundError(f"schema not found: {schema_path}")
    with schema_path.open("r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def _type_ok(value, expected):
    if expected == "string":
        return isinstance(value, str)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "array":
        return isinstance(value, list)
    if expected == "object":
        return isinstance(value, dict)
    return True


def _validate_stage(stage, idx, stage_schema):
    if not isinstance(stage, dict):
        raise ValueError(f"workload stage[{idx}] must be an object")

    required = stage_schema.get("required", [])
    missing = [k for k in required if k not in stage]
    if missing:
        raise ValueError(f"workload stage[{idx}] missing keys: {', '.join(missing)}")

    props = stage_schema.get("properties", {})
    for key, rule in props.items():
        if key not in stage:
            continue
        value = stage[key]
        expected = rule.get("type")
        if expected and not _type_ok(value, expected):
            raise ValueError(f"workload stage[{idx}].{key} expected {expected}")

        if expected == "integer" and "minimum" in rule and value < rule["minimum"]:
            raise ValueError(f"workload stage[{idx}].{key} must be >= {rule['minimum']}")

        if expected == "string" and "pattern" in rule:
            if not re.match(rule["pattern"], value):
                raise ValueError(f"workload stage[{idx}].{key} does not match pattern {rule['pattern']}")


def validate_workload_schema(workload, workload_name):
    if not isinstance(workload, dict):
        raise ValueError(f"workload {workload_name} must be a YAML object")

    schema = _schema_load()
    required = schema.get("required", [])
    props = schema.get("properties", {})

    missing = [k for k in required if k not in workload]
    if missing:
        raise ValueError(f"workload {workload_name} missing keys: {', '.join(missing)}")

    unknown = sorted([k for k in workload.keys() if k not in props])
    if unknown:
        raise ValueError(f"workload {workload_name} has unknown keys: {', '.join(unknown)}")

    for key, rule in props.items():
        if key not in workload:
            continue

        value = workload[key]
        expected = rule.get("type")
        if expected and not _type_ok(value, expected):
            raise ValueError(f"workload {workload_name}.{key} expected {expected}")

        if expected == "string" and "minLength" in rule and len(value) < rule["minLength"]:
            raise ValueError(f"workload {workload_name}.{key} min length is {rule['minLength']}")

        if expected == "integer" and "minimum" in rule and value < rule["minimum"]:
            raise ValueError(f"workload {workload_name}.{key} must be >= {rule['minimum']}")

        if expected == "array" and "minItems" in rule and len(value) < rule["minItems"]:
            raise ValueError(f"workload {workload_name}.{key} must have at least {rule['minItems']} items")

    stage_schema = props.get("stages", {}).get("items", {})
    for idx, stage in enumerate(workload.get("stages", [])):
        _validate_stage(stage, idx, stage_schema)


def normalize_stages(workload):
    stages = workload.get("stages", [])
    normalized = []
    for stage in stages:
        duration = stage.get("duration")
        target = stage.get("target_vus")
        if duration is None or target is None:
            continue
        normalized.append({"duration": duration, "target": int(target)})
    return normalized


def parse_prometheus_metric(text, metric_name):
    pattern = rf"^{re.escape(metric_name)}(?:\{{[^\}}]+\}})?\s+([\d.e+-]+)"
    match = re.search(pattern, text, re.MULTILINE)
    return float(match.group(1)) if match else 0.0


def run_cmd_with_output(command, cwd=None):
    proc = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=False)
    return {
        "command": " ".join(command) if isinstance(command, list) else command,
        "exit_code": proc.returncode,
        "stdout": (proc.stdout or "").strip(),
        "stderr": (proc.stderr or "").strip(),
    }


def detect_compose_digest(lab_dir):
    compose_file = Path(lab_dir) / "docker-compose.yml"
    if not compose_file.exists():
        return ""
    digest = hashlib.sha256(compose_file.read_bytes()).hexdigest()
    return digest


def build_run_id(lab_name, workload_name):
    now = dt.datetime.utcnow().strftime("%Y%m%dT%H%M%SZ")
    return f"{lab_name}__{workload_name}__{now}"


def write_run_metadata(run_dir, lab_name, workload_name, workload, ports):
    metadata = {
        "run_id": run_dir.name,
        "lab": lab_name,
        "workload": workload_name,
        "started_at_utc": dt.datetime.utcnow().isoformat() + "Z",
        "metrics_ports": ports,
        "ws_ports": LAB_WS_PORTS.get(lab_name, [8080]),
        "workload_config": workload,
        "compose_digest_sha256": detect_compose_digest(active_lab_dir),
        "environment": {
            "python": sys.version,
            "platform": sys.platform,
            "host": os.uname().nodename if hasattr(os, "uname") else "unknown",
        },
        "git": run_cmd_with_output(["git", "rev-parse", "HEAD"], cwd=str(BASE_DIR)),
    }

    metadata_path = run_dir / "metadata.json"
    metadata_path.write_text(json.dumps(metadata, indent=2), encoding="utf-8")


def write_legacy_results_link(run_dir, lab_name):
    legacy_csv = RESULTS_DIR / f"{lab_name}_robust_report.csv"
    source_csv = run_dir / "timeseries.csv"
    if source_csv.exists():
        legacy_csv.write_text(source_csv.read_text(encoding="utf-8"), encoding="utf-8")


def flight_recorder(run_dir, ports, stop_event, scrape_interval_seconds=2):
    csv_path = run_dir / "timeseries.csv"

    last_lat_sum = 0.0
    last_lat_count = 0.0
    last_msgs = 0.0
    stuck_counter = 0

    with csv_path.open("w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "timestamp_s",
            "vus",
            "latency_ms",
            "memory_mb",
            "messages_total",
            "dropped_total",
            "errors_total",
        ])

        start_time = time.time()

        while not stop_event.is_set():
            agg_vus = 0.0
            agg_lat_sum = 0.0
            agg_lat_count = 0.0
            agg_mem_bytes = 0.0
            agg_msgs = 0.0
            agg_dropped = 0.0

            for port in ports:
                try:
                    metrics_text = requests.get(f"http://localhost:{port}/metrics", timeout=1).text
                except Exception:
                    continue

                agg_vus += parse_prometheus_metric(metrics_text, "chat_active_connections")
                agg_lat_sum += parse_prometheus_metric(metrics_text, "chat_message_latency_ms_sum")
                agg_lat_count += parse_prometheus_metric(metrics_text, "chat_message_latency_ms_count")
                agg_mem_bytes += parse_prometheus_metric(metrics_text, "chat_memory_bytes")
                agg_msgs += parse_prometheus_metric(metrics_text, "chat_messages_total")
                agg_dropped += parse_prometheus_metric(metrics_text, "chat_dropped_messages_total")

            delta_lat_sum = agg_lat_sum - last_lat_sum
            delta_lat_count = agg_lat_count - last_lat_count
            if delta_lat_count > 0:
                live_latency = delta_lat_sum / delta_lat_count
            elif agg_lat_count > 0:
                live_latency = agg_lat_sum / agg_lat_count
            else:
                live_latency = 0.0

            if agg_msgs == last_msgs and agg_vus > 100:
                stuck_counter += 1
            else:
                stuck_counter = 0

            if stuck_counter >= 10:
                print(f"\n🧨 Benchmark failure: potential deadlock around {agg_vus:.0f} VUs")
                stop_event.set()
                if k6_process:
                    k6_process.terminate()

            last_lat_sum = agg_lat_sum
            last_lat_count = agg_lat_count
            last_msgs = agg_msgs

            elapsed = int(time.time() - start_time)
            mem_mb = agg_mem_bytes / (1024 * 1024)
            writer.writerow([
                elapsed,
                int(agg_vus),
                round(live_latency, 4),
                round(mem_mb, 4),
                round(agg_msgs, 4),
                round(agg_dropped, 4),
                0,
            ])
            f.flush()
            time.sleep(scrape_interval_seconds)


def run_benchmark(lab_name, workload_name="robust_steady"):
    global active_lab_dir, k6_process

    workload = load_workload(workload_name)
    active_lab_dir = LABS_DIR / lab_name
    if not active_lab_dir.exists():
        raise FileNotFoundError(f"lab not found: {lab_name}")

    ports = LAB_METRICS_PORTS.get(lab_name, [8080])
    run_id = build_run_id(lab_name, workload_name)
    run_dir = RAW_RESULTS_DIR / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    write_run_metadata(run_dir, lab_name, workload_name, workload, ports)

    print(f"\n🚀 Running benchmark for {lab_name} with workload {workload_name}...")
    subprocess.run("docker-compose down", shell=True, cwd=active_lab_dir, check=False)
    subprocess.run("docker-compose up --build -d", shell=True, cwd=active_lab_dir, check=False)
    time.sleep(workload.get("warmup_seconds", 5))

    stop_event = threading.Event()
    recorder_thread = threading.Thread(
        target=flight_recorder,
        args=(run_dir, ports, stop_event, workload["scrape_interval_seconds"]),
        daemon=True,
    )
    recorder_thread.start()

    ws_ports = LAB_WS_PORTS.get(lab_name, [8080])
    ws_urls = ",".join([f"ws://localhost:{p}/ws" for p in ws_ports])

    stages_json = json.dumps(normalize_stages(workload))
    user_args = f"--user {os.getuid()}:{os.getgid()}"
    k6_summary = run_dir / "k6_summary.json"

    k6_cmd = (
        f"docker run {user_args} --rm --network host "
        f"-v {BASE_DIR}:/app -w /app grafana/k6 run k6/base.js "
        f"--summary-export=/app/{k6_summary.relative_to(BASE_DIR)} "
        f"--env WS_URLS={ws_urls} "
        f"--env STAGES_JSON='{stages_json}' "
        f"--env MESSAGE_INTERVAL_MS={workload['message_interval_ms']} "
        f"--env WORKLOAD_NAME={workload_name} "
        f"--env DETERMINISTIC_SEED={workload.get('seed', 42)}"
    )

    k6_process = subprocess.Popen(k6_cmd, shell=True)
    k6_process.wait()

    stop_event.set()
    recorder_thread.join(timeout=5)

    subprocess.run("docker-compose down", shell=True, cwd=active_lab_dir, check=False)
    write_legacy_results_link(run_dir, lab_name)

    active_lab_dir = None
    print(f"📊 Benchmark complete. Artifacts: {run_dir}")


# Backward compatibility for existing menu entrypoint.
def run_robust_mode(lab_name):
    run_benchmark(lab_name, workload_name="robust_steady")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 benchmark/orchestrator.py <lab-name> [workload-name]")
        sys.exit(1)

    lab = sys.argv[1]
    workload = sys.argv[2] if len(sys.argv) > 2 else "robust_steady"
    run_benchmark(lab, workload)
