# Lab 05 Benchmark

This benchmark measures the decoupled API-to-worker ingest pipeline introduced in the cloud-native lab.

## What It Runs
- `benchmark/workload.yaml` defines the ingest scenario.
- `benchmark/run.py` orchestrates health checks, Prometheus sampling, and the WebSocket load run.
- `benchmark/plot.py` renders the sampled performance graphs.

## Scenario
- `cloud_native_ingest`: 1 minute at 200 VUs followed by 1 minute at 600 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-05-cloud-native-chat-infrastructure/benchmark/run.py --scenario cloud_native_ingest
```
