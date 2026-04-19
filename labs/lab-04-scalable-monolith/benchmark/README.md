# Lab 04 Benchmark

This benchmark measures the worker-pool version of the monolith and makes queueing pressure visible under higher concurrency.

## What It Runs
- `benchmark/workload.yaml` defines the worker-pool scenario.
- `benchmark/run.py` orchestrates the stack, Prometheus sampling, and load execution.
- `benchmark/plot.py` turns the sampled time series into run and suite graphs.

## Scenario
- `scaling_monolith`: 1 minute at 200 VUs followed by 1 minute at 500 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-04-scalable-monolith/benchmark/run.py --scenario scaling_monolith
```
