# Lab 07 Benchmark

This benchmark measures the cost of presence synchronization and other high-frequency ephemeral state updates.

## What It Runs
- `benchmark/workload.yaml` defines the presence scaling scenario.
- `benchmark/run.py` handles startup, Prometheus sampling, and the WebSocket load run.
- `benchmark/plot.py` produces the time-series graphs for run and suite views.

## Scenario
- `presence_scaling`: 1 minute at 100 VUs followed by 1 minute at 400 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-07-real-time-presence-and-delivery/benchmark/run.py --scenario presence_scaling
```
