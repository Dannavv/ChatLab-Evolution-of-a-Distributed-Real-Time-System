import argparse
import datetime as dt
import json
import os
import signal
import subprocess
import sys
import threading
import time
from pathlib import Path

import requests
import yaml

from plot import generate_run_graphs, generate_suite_graphs

LAB_DIR = Path(__file__).resolve().parent.parent
ROOT_DIR = LAB_DIR.parent.parent
RESULTS_ROOT = LAB_DIR / 'benchmark' / 'results'
WORKLOAD_FILE = LAB_DIR / 'benchmark' / 'workload.yaml'

k6_process = None

def load_workload():
    if not WORKLOAD_FILE.exists():
        raise FileNotFoundError(f'workload manifest missing: {WORKLOAD_FILE}')
    with WORKLOAD_FILE.open('r', encoding='utf-8') as f:
        return yaml.safe_load(f)

def read_metric_value(text, metric_name):
    import re
    pattern = rf'^{metric_name}(?:\{{[^}}]+\}})?\s+([\d.e+-]+)'
    match = re.search(pattern, text, re.MULTILINE)
    return float(match.group(1)) if match else 0.0

def run_command(command, cwd=None, check=False):
    return subprocess.run(command, cwd=cwd, shell=True, check=check)

def cleanup(sig=None, frame=None):
    global k6_process
    if k6_process and k6_process.poll() is None:
        k6_process.terminate()
    run_command('docker-compose down', cwd=LAB_DIR)
    print('\nStopped lab 02 benchmark.')
    sys.exit(0)

signal.signal(signal.SIGINT, cleanup)
signal.signal(signal.SIGTERM, cleanup)

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

def run_sampler(metrics_url, output_path, stop_event, scrape_interval_seconds):
    output_path.parent.mkdir(parents=True, exist_ok=True)
    start = time.time()
    last_latency_sum = 0.0
    last_latency_count = 0.0
    last_db_sum = 0.0
    last_db_count = 0.0

    with output_path.open('w', encoding='utf-8') as f:
        # sql_latency_ms corresponds to chat_db_query_duration_ms in Prometheus
        f.write('timestamp_s,vus,latency_ms,db_latency_ms,memory_mb,messages_total,dropped_total\n')
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

            delta_sum = latency_sum - last_latency_sum
            delta_count = latency_count - last_latency_count
            latency_ms = delta_sum / delta_count if delta_count > 0 else (latency_sum / latency_count if latency_count > 0 else 0.0)

            delta_db_sum = db_sum - last_db_sum
            delta_db_count = db_count - last_db_count
            db_latency_ms = delta_db_sum / delta_db_count if delta_db_count > 0 else (db_sum / db_count if db_count > 0 else 0.0)

            elapsed = int(time.time() - start)
            f.write(
                f'{elapsed},{int(active_connections)},{latency_ms:.4f},{db_latency_ms:.4f},'
                f'{memory_bytes / (1024 * 1024):.4f},{messages_total:.4f},{dropped_total:.4f}\n'
            )
            f.flush()
            last_latency_sum = latency_sum
            last_latency_count = latency_count
            last_db_sum = db_sum
            last_db_count = db_count
            time.sleep(scrape_interval_seconds)

def run_scenario(scenario_name, scenario, workload):
    global k6_process
    run_id = f"lab02__{scenario_name}__{dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ')}"
    run_dir = RESULTS_ROOT / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    metadata = {
        'run_id': run_id,
        'lab': 'lab-02-persistence-layer',
        'scenario': scenario_name,
        'started_at_utc': dt.datetime.now(dt.timezone.utc).isoformat().replace('+00:00', 'Z'),
        'workload': workload
    }
    (run_dir / 'metadata.json').write_text(json.dumps(metadata, indent=2))

    print(f'\n🚀 Running Lab 02: {scenario_name}')
    run_command('docker-compose down', cwd=LAB_DIR)
    run_command('docker-compose up --build -d', cwd=LAB_DIR, check=True)
    wait_for_health(workload['health_url'])

    stop_event = threading.Event()
    sampler_thread = threading.Thread(
        target=run_sampler,
        args=(workload['metrics_url'], run_dir / 'timeseries.csv', stop_event, workload['scrape_interval_seconds']),
        daemon=True,
    )
    sampler_thread.start()

    normalized_stages = [{'duration': s['duration'], 'target': int(s['target_vus'])} for s in scenario.get('stages', [])]
    stages_json = json.dumps(normalized_stages)
    
    command = (
        'docker run --rm --network host '
        f'-v {ROOT_DIR}:/app -w /app grafana/k6 run k6/base.js '
        f"--env WS_URLS={workload['ws_url']} "
        f"--env STAGES_JSON='{stages_json}' "
        f"--env MESSAGE_INTERVAL_MS={scenario['message_interval_ms']} "
    )

    k6_process = subprocess.Popen(command, shell=True)
    k6_process.wait()
    stop_event.set()
    sampler_thread.join(timeout=10)
    run_command('docker-compose down', cwd=LAB_DIR)
    
    generate_run_graphs(run_dir)
    return run_dir

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--scenario', default=None)
    parser.add_argument('--all', action='store_true')
    args = parser.parse_args()

    workload = load_workload()
    scenarios = workload.get('scenarios', {})
    RESULTS_ROOT.mkdir(parents=True, exist_ok=True)
    
    selected = scenarios.items() if (args.all or not args.scenario) else [(args.scenario, scenarios.get(args.scenario))]
    
    for scenario_name, scenario in selected:
        if scenario:
            run_scenario(scenario_name, scenario, workload)
    
    generate_suite_graphs(RESULTS_ROOT)
    print('\n✅ Lab 02 benchmark complete.')