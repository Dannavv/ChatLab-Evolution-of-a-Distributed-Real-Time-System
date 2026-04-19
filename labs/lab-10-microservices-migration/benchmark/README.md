# Lab 10 Benchmark

This benchmark measures the gateway-plus-services mesh created in the microservices migration lab.

## What It Runs
- `benchmark/workload.yaml` defines the mesh scenario.
- `benchmark/run.py` starts the stack, samples Prometheus, and runs the WebSocket load execution.
- `benchmark/plot.py` renders run and suite graphs from the sampled outputs.

## Scenario
- `microservices_mesh`: 1 minute at 100 VUs followed by 1 minute at 400 VUs with a 1000 ms message interval.

## Outputs
Each run writes into `benchmark/results/<run_id>/`:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

## Run
```bash
python3 labs/lab-10-microservices-migration/benchmark/run.py --scenario microservices_mesh
```
