# Lab 02 Benchmark

This benchmark measures the persistence-layer version of the chat service and focuses on the cost of durable writes compared with the Lab 01 in-memory baseline.

## What It Runs
- `benchmark/workload.yaml` defines the Lab 02 scenarios.
- `benchmark/run.py` starts the stack, samples Prometheus, runs the load driver, and writes per-run artifacts.
- `benchmark/plot.py` renders per-run and suite graphs.
- `benchmark/compare.py` generates the Lab 01 vs Lab 02 overlay and the Lab 02 latency-breakdown asset.

## Scenarios
- `comparison_standard`: fair cross-lab comparison scenario shared with Lab 01.
- `persistence_standard`: steady durable load at 100 VUs.
- `persistence_stress`: higher concurrency durable load at 500 VUs.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `k6_summary.json`
- `benchmark_summary.json`
- `graphs/*.png`

Permanent README-facing assets are written under `assets/benchmarks/`.

## Run
From the repository root:
```bash
python3 labs/lab-02-persistence-layer/benchmark/run.py --scenario comparison_standard
```

To regenerate the comparison assets:
```bash
python3 labs/lab-02-persistence-layer/benchmark/compare.py
```
