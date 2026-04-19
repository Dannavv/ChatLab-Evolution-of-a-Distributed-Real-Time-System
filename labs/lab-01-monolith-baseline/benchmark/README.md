# Lab 01 Benchmark

This benchmark lives inside the Lab 01 folder so the monolith has its own dedicated test harness.

## What it runs
- `benchmark/workload.yaml` defines Lab 01 scenarios.
- `k6/lab01.js` drives the WebSocket load.
- `benchmark/run.py` starts the lab, samples Prometheus metrics, runs k6, and writes outputs.
- `benchmark/plot.py` renders a large set of graphs from every run.

## Outputs
Each run produces:
- `timeseries.csv`
- `k6_summary.json`
- `metadata.json`
- `report.md`
- `graphs/*.png`

The suite-level comparison is written under `benchmark/results/suite/`.

## Run
From the repository root:
```bash
python3 labs/lab-01-monolith-baseline/benchmark/run.py --all
```

To run one scenario:
```bash
python3 labs/lab-01-monolith-baseline/benchmark/run.py --scenario saturation
```
