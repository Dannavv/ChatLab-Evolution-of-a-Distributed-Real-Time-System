# Lab 06 Benchmark

This benchmark exercises the resilience layer under load so circuit-breaker and retry behavior can be observed in the time series.

## What It Runs
- `benchmark/workload.yaml` defines the resilience stress scenario.
- `benchmark/run.py` manages startup, Prometheus scraping, and the load execution.
- `benchmark/plot.py` renders run and suite graphs from the captured samples.

## Scenario
- `resilience_stress`: 1 minute at 200 VUs followed by 1 minute at 800 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-06-chaos-and-resilience/benchmark/run.py --scenario resilience_stress
```
