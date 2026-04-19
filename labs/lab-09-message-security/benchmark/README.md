# Lab 09 Benchmark

This benchmark measures the overhead introduced by encryption and key-management behavior in the secure messaging lab.

## What It Runs
- `benchmark/workload.yaml` defines the security overhead scenario.
- `benchmark/run.py` coordinates startup, metrics scraping, and the WebSocket load run.
- `benchmark/plot.py` renders the performance graphs from sampled time-series data.

## Scenario
- `security_overhead`: 1 minute at 100 VUs followed by 1 minute at 400 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-09-message-security/benchmark/run.py --scenario security_overhead
```
