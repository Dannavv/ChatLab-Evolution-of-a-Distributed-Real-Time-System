import argparse
import csv
import datetime as dt
import json
import math
import signal
import subprocess
import sys
import threading
import time
from pathlib import Path

import requests
import yaml

from shared.benchmark.plotting import generate_run_graphs, generate_suite_graphs
from shared.benchmark.report import build_comparison_artifacts


k6_process = None

REQUIRED_WORKLOAD_KEYS = [
    'benchmark_contract_version',
    'name',
    'lab',
    'health_url',
    'metrics_url',
    'ws_url',
    'warmup_seconds',
    'scrape_interval_seconds',
    'summary_interval_seconds',
    'message_payload_bytes',
    'room_id',
    'results_dir',
    'comparison_profile',
    'workload_model',
    'metrics_contract',
    'traceability',
    'consistency_model',
    'failure_model',
    'routing_strategy',
    'observability',
    'cost_model',
    'synthesis',
    'scenarios',
]


def read_metric_value(text, metric_name):
    import re
    pattern = rf'^{metric_name}(?:\{{[^}}]+\}})?\s+([\d.e+-]+)'
    match = re.search(pattern, text, re.MULTILINE)
    return float(match.group(1)) if match else 0.0


def run_command(command, cwd=None, check=False):
    return subprocess.run(command, cwd=cwd, shell=True, check=check)


def wait_for_health(url, timeout_seconds=90):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            response = requests.get(url, timeout=2)
            if response.ok:
                return True
        except Exception:
            pass
        time.sleep(1)
    raise TimeoutError(f'health check timed out: {url}')


def percentile(values, p):
    if not values:
        return 0.0
    ordered = sorted(values)
    idx = (len(ordered) - 1) * p
    lower = math.floor(idx)
    upper = math.ceil(idx)
    if lower == upper:
        return float(ordered[lower])
    weight = idx - lower
    return float(ordered[lower] * (1 - weight) + ordered[upper] * weight)


def build_latency_histogram(latencies):
    buckets = [
        (0, 1),
        (1, 5),
        (5, 10),
        (10, 25),
        (25, 50),
        (50, 100),
        (100, None),
    ]
    lines = []
    total = len(latencies) or 1
    for low, high in buckets:
        if high is None:
            count = sum(1 for value in latencies if value >= low)
            label = f'{low:>3}ms+'
        else:
            count = sum(1 for value in latencies if low <= value < high)
            label = f'{low:>3}-{high:<3}ms'
        bar = '#' * max(1, int((count / total) * 30)) if count else ''
        lines.append(f'{label} | {bar} {count}')
    return lines


def load_timeseries_rows(csv_path):
    with Path(csv_path).open('r', encoding='utf-8') as handle:
        return list(csv.DictReader(handle))


def parse_k6_counts(summary_path):
    summary_path = Path(summary_path)
    if not summary_path.exists():
        return {}
    try:
        payload = json.loads(summary_path.read_text(encoding='utf-8'))
    except Exception:
        return {}

    metrics = payload.get('metrics', {})

    def metric_count(name):
        values = metrics.get(name, {}).get('values', {})
        return int(values.get('count', 0) or 0)

    return {
        'sent': metric_count('chat_messages_sent'),
        'received': metric_count('chat_messages_received'),
        'duplicates': metric_count('chat_duplicate_messages'),
    }


def _ensure_workload_contract(workload, workload_file):
    missing = [key for key in REQUIRED_WORKLOAD_KEYS if key not in workload]
    if missing:
        raise ValueError(f'workload contract missing keys in {workload_file}: {", ".join(missing)}')


def load_workload(lab_dir):
    workload_file = Path(lab_dir) / 'benchmark' / 'workload.yaml'
    if not workload_file.exists():
        raise FileNotFoundError(f'workload manifest missing: {workload_file}')
    with workload_file.open('r', encoding='utf-8') as handle:
        workload = yaml.safe_load(handle)
    _ensure_workload_contract(workload, workload_file)
    return workload


def write_run_summary(run_dir):
    rows = load_timeseries_rows(Path(run_dir) / 'timeseries.csv')
    latencies = [float(row['latency_ms']) for row in rows if float(row['latency_ms']) > 0]
    db_latencies = [float(row.get('db_latency_ms', 0) or 0) for row in rows if float(row.get('db_latency_ms', 0) or 0) > 0]
    throughputs = [float(row.get('throughput_msgs_s', 0) or 0) for row in rows]
    final_row = rows[-1] if rows else {}
    k6_counts = parse_k6_counts(Path(run_dir) / 'k6_summary.json')

    sent = k6_counts.get('sent', 0)
    received = k6_counts.get('received', 0)
    duplicates = k6_counts.get('duplicates', 0)
    error_rate_pct = float(final_row.get('error_rate_pct', 0) or 0)
    delivery_ratio_pct = (received / sent * 100.0) if sent > 0 else 0.0

    summary = {
        'latency_ms': {
            'p50': round(percentile(latencies, 0.50), 3),
            'p90': round(percentile(latencies, 0.90), 3),
            'p95': round(percentile(latencies, 0.95), 3),
            'p99': round(percentile(latencies, 0.99), 3),
        },
        'db_latency_ms': {
            'p50': round(percentile(db_latencies, 0.50), 3),
            'p90': round(percentile(db_latencies, 0.90), 3),
            'p95': round(percentile(db_latencies, 0.95), 3),
            'p99': round(percentile(db_latencies, 0.99), 3),
        },
        'throughput_msgs_s': {
            'avg': round(sum(throughputs) / len(throughputs), 3) if throughputs else 0.0,
            'peak': round(max(throughputs), 3) if throughputs else 0.0,
        },
        'totals': {
            'messages_total': int(float(final_row.get('messages_total', 0) or 0)),
            'dropped_total': int(float(final_row.get('dropped_total', 0) or 0)),
            'db_errors_total': int(float(final_row.get('db_errors_total', 0) or 0)),
            'sent': sent,
            'received': received,
            'duplicates': duplicates,
        },
        'reliability': {
            'error_rate_pct': round(error_rate_pct, 3),
            'delivery_ratio_pct': round(delivery_ratio_pct, 3),
            'duplicate_ratio_pct': round((duplicates / sent * 100.0), 3) if sent > 0 else 0.0,
        },
        'latency_histogram': build_latency_histogram(latencies),
    }
    Path(run_dir, 'benchmark_summary.json').write_text(json.dumps(summary, indent=2), encoding='utf-8')

    print('\nBenchmark summary')
    print(
        'Latency p50/p90/p95/p99: '
        f"{summary['latency_ms']['p50']:.2f} / {summary['latency_ms']['p90']:.2f} / "
        f"{summary['latency_ms']['p95']:.2f} / {summary['latency_ms']['p99']:.2f} ms"
    )
    if db_latencies:
        print(
            'DB write p50/p90/p95/p99: '
            f"{summary['db_latency_ms']['p50']:.2f} / {summary['db_latency_ms']['p90']:.2f} / "
            f"{summary['db_latency_ms']['p95']:.2f} / {summary['db_latency_ms']['p99']:.2f} ms"
        )
    print(
        'Throughput avg/peak: '
        f"{summary['throughput_msgs_s']['avg']:.2f} / {summary['throughput_msgs_s']['peak']:.2f} msgs/s"
    )
    print(
        'Reliability error/delivery/duplicate: '
        f"{summary['reliability']['error_rate_pct']:.2f}% / "
        f"{summary['reliability']['delivery_ratio_pct']:.2f}% / "
        f"{summary['reliability']['duplicate_ratio_pct']:.2f}%"
    )
    print(
        'Sanity check sent/received/server-count/duplicates: '
        f"{summary['totals']['sent']} / {summary['totals']['received']} / "
        f"{summary['totals']['messages_total']} / {summary['totals']['duplicates']}"
    )
    print('Latency histogram')
    for line in summary['latency_histogram']:
        print(f'  {line}')


def run_sampler(metrics_url, output_path, stop_event, scrape_interval_seconds, summary_interval_seconds):
    output_path = Path(output_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    start = time.time()
    last_latency_sum = 0.0
    last_latency_count = 0.0
    last_db_sum = 0.0
    last_db_count = 0.0
    last_messages_total = 0.0
    last_summary_tick = -1

    with output_path.open('w', encoding='utf-8') as handle:
        handle.write('timestamp_s,vus,latency_ms,db_latency_ms,memory_mb,messages_total,dropped_total,db_errors_total,error_rate_pct,throughput_msgs_s\n')
        while not stop_event.is_set():
            try:
                text = requests.get(metrics_url, timeout=2).text
            except Exception:
                time.sleep(scrape_interval_seconds)
                continue

            active_connections = read_metric_value(text, 'chat_active_connections')
            latency_sum = read_metric_value(text, 'chat_message_latency_ms_sum')
            latency_count = read_metric_value(text, 'chat_message_latency_ms_count')
            db_sum = read_metric_value(text, 'chat_db_query_duration_ms_sum')
            db_count = read_metric_value(text, 'chat_db_query_duration_ms_count')
            memory_bytes = read_metric_value(text, 'chat_memory_bytes')
            messages_total = read_metric_value(text, 'chat_messages_total')
            dropped_total = read_metric_value(text, 'chat_dropped_messages_total')
            db_errors_total = read_metric_value(text, 'chat_db_errors_total')

            delta_sum = latency_sum - last_latency_sum
            delta_count = latency_count - last_latency_count
            latency_ms = delta_sum / delta_count if delta_count > 0 else (latency_sum / latency_count if latency_count > 0 else 0.0)

            delta_db_sum = db_sum - last_db_sum
            delta_db_count = db_count - last_db_count
            db_latency_ms = delta_db_sum / delta_db_count if delta_db_count > 0 else (db_sum / db_count if db_count > 0 else 0.0)
            throughput_msgs_s = max(0.0, (messages_total - last_messages_total) / max(scrape_interval_seconds, 1))
            error_rate_pct = ((dropped_total + db_errors_total) / messages_total * 100.0) if messages_total > 0 else 0.0

            elapsed = int(time.time() - start)
            handle.write(
                f'{elapsed},{int(active_connections)},{latency_ms:.4f},{db_latency_ms:.4f},'
                f'{memory_bytes / (1024 * 1024):.4f},{messages_total:.4f},{dropped_total:.4f},'
                f'{db_errors_total:.4f},{error_rate_pct:.4f},{throughput_msgs_s:.4f}\n'
            )
            handle.flush()
            last_latency_sum = latency_sum
            last_latency_count = latency_count
            last_db_sum = db_sum
            last_db_count = db_count
            last_messages_total = messages_total

            summary_tick = elapsed // max(summary_interval_seconds, 1)
            if elapsed > 0 and summary_tick != last_summary_tick and elapsed % max(summary_interval_seconds, 1) == 0:
                print(
                    f'[t+{elapsed:>3}s] vus={int(active_connections):>4} '
                    f'tput={throughput_msgs_s:>7.2f} msgs/s '
                    f'lat={latency_ms:>7.2f} ms '
                    f'db={db_latency_ms:>7.2f} ms '
                    f'err={error_rate_pct:>6.2f}%'
                )
                last_summary_tick = summary_tick
            time.sleep(scrape_interval_seconds)


def run_scenario(lab_dir, scenario_name, scenario, workload):
    global k6_process

    lab_dir = Path(lab_dir)
    lab_name = lab_dir.name
    root_dir = lab_dir.parent.parent
    results_root = lab_dir / workload.get('results_dir', 'benchmark/results')
    results_root.mkdir(parents=True, exist_ok=True)

    lab_id_num = lab_name.split('-')[1]
    run_id = f"lab{lab_id_num}__{scenario_name}__{dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ')}"
    run_dir = results_root / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    metadata = {
        'run_id': run_id,
        'lab': lab_name,
        'scenario': scenario_name,
        'description': scenario.get('description', ''),
        'started_at_utc': dt.datetime.now(dt.timezone.utc).isoformat().replace('+00:00', 'Z'),
        'benchmark_contract_version': workload['benchmark_contract_version'],
        'comparison_profile': workload['comparison_profile'],
        'workload_model': workload['workload_model'],
        'metrics_contract': workload['metrics_contract'],
        'traceability': workload['traceability'],
        'consistency_model': workload['consistency_model'],
        'failure_model': workload['failure_model'],
        'routing_strategy': workload['routing_strategy'],
        'observability': workload['observability'],
        'cost_model': workload['cost_model'],
        'synthesis': workload['synthesis'],
        'workload': workload,
    }
    (run_dir / 'metadata.json').write_text(json.dumps(metadata, indent=2), encoding='utf-8')

    print(f'\n🚀 Starting {lab_name.upper()}: {scenario_name}')
    run_command('docker-compose down', cwd=lab_dir)
    run_command('docker-compose up --build -d', cwd=lab_dir, check=True)
    time.sleep(workload.get('warmup_seconds', 0))
    wait_for_health(workload['health_url'])

    stop_event = threading.Event()
    sampler_thread = threading.Thread(
        target=run_sampler,
        args=(
            workload['metrics_url'],
            run_dir / 'timeseries.csv',
            stop_event,
            workload['scrape_interval_seconds'],
            workload.get('summary_interval_seconds', 5),
        ),
        daemon=True,
    )
    sampler_thread.start()

    normalized_stages = [{'duration': stage['duration'], 'target': int(stage['target_vus'])} for stage in scenario.get('stages', [])]
    stages_json = json.dumps(normalized_stages)
    k6_summary_path = run_dir / 'k6_summary.json'

    command = (
        'docker run --rm --network host '
        f'-v {root_dir}:/app -w /app grafana/k6 run k6/base.js '
        f'--summary-export=/app/{k6_summary_path.relative_to(root_dir)} '
        f"--env WS_URLS={workload['ws_url']} "
        f"--env STAGES_JSON='{stages_json}' "
        f"--env MESSAGE_INTERVAL_MS={scenario['message_interval_ms']} "
        f"--env TARGET_MESSAGE_BYTES={workload.get('message_payload_bytes', 0)} "
        f"--env ROOM_ID={workload.get('room_id', 'benchmark-room')} "
    )

    k6_process = subprocess.Popen(command, shell=True)
    k6_process.wait()
    stop_event.set()
    sampler_thread.join(timeout=10)
    run_command('docker-compose down', cwd=lab_dir)

    generate_run_graphs(run_dir)
    write_run_summary(run_dir)
    build_comparison_artifacts()
    return run_dir


def main(lab_dir):
    lab_dir = Path(lab_dir)
    results_root = lab_dir / 'benchmark' / 'results'

    def cleanup(sig=None, frame=None):
        global k6_process
        if k6_process and k6_process.poll() is None:
            k6_process.terminate()
        run_command('docker-compose down', cwd=lab_dir)
        print(f'\nStopped {lab_dir.name} benchmark.')
        sys.exit(0)

    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)

    parser = argparse.ArgumentParser()
    parser.add_argument('--scenario', default=None)
    parser.add_argument('--all', action='store_true')
    args = parser.parse_args()

    workload = load_workload(lab_dir)
    scenarios = workload.get('scenarios', {})
    results_root.mkdir(parents=True, exist_ok=True)

    selected = scenarios.items() if (args.all or not args.scenario) else [(args.scenario, scenarios.get(args.scenario))]
    for scenario_name, scenario in selected:
        if scenario:
            run_scenario(lab_dir, scenario_name, scenario, workload)

    generate_suite_graphs(results_root)
    build_comparison_artifacts()
    print(f'\n✅ {lab_dir.name.upper()} benchmark complete.')
