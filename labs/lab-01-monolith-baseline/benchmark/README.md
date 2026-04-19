# Lab 01 Benchmark

This benchmark establishes the monolith baseline and provides the control-group data for the rest of the curriculum.

## What It Runs
- `benchmark/workload.yaml` defines Lab 01 scenarios.
- `benchmark/run.py` starts the lab, samples Prometheus, runs the load driver, and writes outputs.
- `benchmark/plot.py` renders run and suite graphs from the sampled data.

## Scenarios
- `comparison_standard`: fair cross-lab comparison scenario shared with Lab 02.
- `baseline`: steady load to capture the operating curve.
- `saturation`: sustained pressure to expose mutex and broadcast bottlenecks.
- `spike_recovery`: abrupt spikes to observe tail latency and rebound.

## Outputs
Each run produces:
- `timeseries.csv`
- `k6_summary.json`
- `metadata.json`
- `benchmark_summary.json`
- `graphs/*.png`

The suite-level comparison is written under `benchmark/results/suite/`.

## Run
From the repository root:
```bash
python3 labs/lab-01-monolith-baseline/benchmark/run.py --scenario comparison_standard
```

To run one scenario:
```bash
python3 labs/lab-01-monolith-baseline/benchmark/run.py --scenario saturation
```
