# Lab 11 Benchmark

This benchmark validates the deployable blueprint stack under the same fair-comparison contract used by the rest of the repo.

## What It Runs
- `benchmark/workload.yaml` defines the comparable and validation scenarios.
- `benchmark/run.py` starts the stack, samples Prometheus, and runs the WebSocket load generator.
- The shared plotting and reporting pipeline updates the repo-wide comparison artifacts.

## Scenarios
- `comparison_standard`: fair cross-lab comparison scenario
- `blueprint_validation`: modest capstone steady-load validation

## Run

```bash
python3 scripts/chatlab.py bench lab-11-production-grade-blueprint --scenario comparison_standard
```
