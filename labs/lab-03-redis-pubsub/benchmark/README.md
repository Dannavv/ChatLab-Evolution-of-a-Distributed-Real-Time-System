# Lab 03 Benchmark

This benchmark evaluates the Redis-backed distributed fan-out path introduced in Lab 03.

## What It Runs
- `benchmark/workload.yaml` defines the Redis scaling scenario.
- `benchmark/run.py` starts the lab, samples Prometheus, runs the WebSocket load driver, and writes per-run artifacts.
- `benchmark/plot.py` renders run-level and suite-level graphs from `timeseries.csv`.

## Scenario
- `redis_scaling`: 1 minute at 100 VUs followed by 1 minute at 300 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-03-redis-pubsub/benchmark/run.py --scenario redis_scaling
```
