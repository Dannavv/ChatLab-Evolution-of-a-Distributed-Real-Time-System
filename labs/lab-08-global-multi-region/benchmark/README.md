# Lab 08 Benchmark

This benchmark measures the regional and inter-region behavior of the global multi-region design.

## What It Runs
- `benchmark/workload.yaml` defines the global synchronization scenario.
- `benchmark/run.py` orchestrates the stack, metrics sampling, and the WebSocket load execution.
- `benchmark/plot.py` renders run and suite graphs from `timeseries.csv`.

## Scenario
- `global_scaling`: 1 minute at 100 VUs followed by 1 minute at 400 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-08-global-multi-region/benchmark/run.py --scenario global_scaling
```
