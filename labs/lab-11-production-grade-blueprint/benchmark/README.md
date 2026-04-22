# Lab 11 Benchmark

This benchmark validates the deployable blueprint stack under the same fair-comparison contract used by the rest of the repo.

## 🧪 What It Runs
- **Workload:** `benchmark/workload.yaml` defines the comparable and validation scenarios.
- **Chaos Testing:** Supports the `--chaos` flag to inject service failures mid-run.
- **Metrics:** Automatically samples Prometheus and calculates the **Recovery Time Objective (RTO)**.
- **Visuals:** Generates latency, throughput, and reliability graphs.

## 📈 Scenarios
- `comparison_standard`: Fair cross-lab comparison scenario.
- `blueprint_validation`: Modest capstone steady-load validation.

## 🚀 Execution

The recommended way to run this is via the root `Makefile`:

```bash
# Standard benchmark
make bench LAB=lab-11-production-grade-blueprint

# Chaos-injected benchmark (tests Circuit Breakers and Recovery)
make bench LAB=lab-11-production-grade-blueprint chaos=true
```

Or using the raw CLI:

```bash
python3 scripts/chatlab.py bench lab-11-production-grade-blueprint --scenario comparison_standard --chaos
```
