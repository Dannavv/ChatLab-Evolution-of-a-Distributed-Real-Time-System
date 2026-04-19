import csv
import json
from pathlib import Path

import yaml


ROOT_DIR = Path(__file__).resolve().parents[2]
LABS_DIR = ROOT_DIR / 'labs'
RESULTS_DIR = ROOT_DIR / 'results'
COMPARISON_MD = RESULTS_DIR / 'comparison.md'
COMPARISON_JSON = RESULTS_DIR / 'comparison.json'


def _load_yaml(path):
    with Path(path).open('r', encoding='utf-8') as handle:
        return yaml.safe_load(handle)


def _load_json(path, fallback=None):
    try:
        return json.loads(Path(path).read_text(encoding='utf-8'))
    except Exception:
        return fallback if fallback is not None else {}


def _latest_run_for(results_root, scenario_name):
    if not results_root.exists():
        return None
    candidates = []
    for entry in results_root.iterdir():
        if not entry.is_dir():
            continue
        metadata = _load_json(entry / 'metadata.json')
        if metadata.get('scenario') != scenario_name:
            continue
        candidates.append(entry)
    if not candidates:
        return None
    return max(candidates, key=lambda item: item.stat().st_mtime)


def _read_last_row(csv_path):
    if not Path(csv_path).exists():
        return {}
    with Path(csv_path).open('r', encoding='utf-8') as handle:
        rows = list(csv.DictReader(handle))
    return rows[-1] if rows else {}


def _progression_row(lab_dir):
    workload = _load_yaml(lab_dir / 'benchmark' / 'workload.yaml')
    comparison_profile = workload.get('comparison_profile', {})
    scenario_name = comparison_profile.get('comparable_scenario', 'comparison_standard')
    latest_run = _latest_run_for(lab_dir / 'benchmark' / 'results', scenario_name)

    base = {
        'lab': lab_dir.name,
        'scenario': scenario_name,
        'workload': workload,
        'run_dir': str(latest_run) if latest_run else None,
        'status': 'PENDING COMPARABLE RUN',
    }
    if latest_run is None:
        return base

    summary = _load_json(latest_run / 'benchmark_summary.json')
    metadata = _load_json(latest_run / 'metadata.json')
    final_row = _read_last_row(latest_run / 'timeseries.csv')
    base.update({
        'summary': summary,
        'metadata': metadata,
        'final_row': final_row,
        'status': 'OK',
    })
    return base


def _fmt_number(value, digits=2):
    try:
        return f'{float(value):.{digits}f}'
    except Exception:
        return '-'


def _status_line(row):
    if row['status'] != 'OK':
        return row['status']
    summary = row.get('summary', {})
    reliability = summary.get('reliability', {})
    delivery = reliability.get('delivery_ratio_pct')
    if delivery is None:
        return 'OK'
    return f"OK ({_fmt_number(delivery)}% delivered)"


def build_comparison_artifacts():
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    rows = [
        _progression_row(lab_dir)
        for lab_dir in sorted(LABS_DIR.iterdir())
        if lab_dir.is_dir() and (lab_dir / 'benchmark' / 'workload.yaml').exists()
    ]

    output_rows = []
    md_lines = [
        '# Benchmark Comparison Report',
        '',
        'This report compares the latest run of each lab\'s shared `comparison_standard` scenario, using the repo-wide benchmark contract.',
        '',
        '| Lab | Scenario | P95 Latency (ms) | Avg Throughput (msgs/s) | Error Rate (%) | Consistency | Routing | Failure Focus | Cost Axis | Status |',
        '|---|---|---|---|---|---|---|---|---|---|',
    ]

    for row in rows:
        workload = row['workload']
        summary = row.get('summary', {})
        latency = summary.get('latency_ms', {})
        throughput = summary.get('throughput_msgs_s', {})
        reliability = summary.get('reliability', {})
        consistency = workload.get('consistency_model', {}).get('target', '-')
        routing = workload.get('routing_strategy', {}).get('policy', '-')
        failure_focus = workload.get('failure_model', {}).get('focus', '-')
        cost_axis = workload.get('cost_model', {}).get('dominant_axis', '-')

        md_lines.append(
            f"| {row['lab']} | {row['scenario']} | {_fmt_number(latency.get('p95'))} | "
            f"{_fmt_number(throughput.get('avg'))} | {_fmt_number(reliability.get('error_rate_pct'))} | "
            f"{consistency} | {routing} | {failure_focus} | {cost_axis} | {_status_line(row)} |"
        )

        output_rows.append({
            'lab': row['lab'],
            'scenario': row['scenario'],
            'status': row['status'],
            'run_dir': row.get('run_dir'),
            'latency_ms': latency,
            'throughput_msgs_s': throughput,
            'reliability': reliability,
            'consistency_model': workload.get('consistency_model', {}),
            'routing_strategy': workload.get('routing_strategy', {}),
            'failure_model': workload.get('failure_model', {}),
            'cost_model': workload.get('cost_model', {}),
            'synthesis': workload.get('synthesis', {}),
        })

    md_lines.extend([
        '',
        '## Progression Summary',
        '',
        '| Lab | Scaling Inflection | Traceability | Observability |',
        '|---|---|---|---|',
    ])

    for row in rows:
        workload = row['workload']
        md_lines.append(
            f"| {row['lab']} | {workload.get('synthesis', {}).get('scaling_inflection', '-')} | "
            f"{workload.get('traceability', {}).get('contract', '-')} | "
            f"{workload.get('observability', {}).get('baseline', '-')} |"
        )

    COMPARISON_MD.write_text('\n'.join(md_lines) + '\n', encoding='utf-8')
    COMPARISON_JSON.write_text(json.dumps({'generated_rows': output_rows}, indent=2), encoding='utf-8')
    return COMPARISON_MD
