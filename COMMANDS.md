# ChatLab Control Commands

This repo now uses one standard control script for setup, benchmarking, observability, reporting, and failure injection.

## Core Workflow

```bash
python3 scripts/chatlab.py list
python3 scripts/chatlab.py up lab-01-monolith-baseline
python3 scripts/chatlab.py observe lab-01-monolith-baseline
python3 scripts/chatlab.py bench lab-01-monolith-baseline --scenario comparison_standard
python3 scripts/chatlab.py down lab-01-monolith-baseline
```

## Benchmarking

Run the fair-comparison suite across all labs:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard
```

Include Lab 11 in the suite when you want capstone evidence:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard --include-blueprint
```

Run Lab 11 on demand:

```bash
python3 scripts/chatlab.py bench lab-11-production-grade-blueprint --scenario comparison_standard
```

Rebuild the aggregate comparison report:

```bash
python3 scripts/chatlab.py report
```

Run local validation gates:

```bash
python3 scripts/chatlab.py validate
python3 scripts/chatlab.py validate --kind workloads
python3 scripts/chatlab.py validate --kind results
python3 scripts/chatlab.py validate --kind slos
```

The interactive benchmark menu still exists:

```bash
python3 main.py
```

## Logs And Observability

```bash
python3 scripts/chatlab.py observe lab-06-chaos-and-resilience
python3 scripts/chatlab.py logs lab-06-chaos-and-resilience --follow
```

## Failure Injection

```bash
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience kill chat-worker
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience delay redis --latency-ms 300 --jitter-ms 50
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience heal redis
```
