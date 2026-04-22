# ChatLab Control Guide

This repository uses a standardized control system for environment setup, benchmarking, observability, and chaos testing.

## 🚀 Recommended Workflow (Makefile)

The `Makefile` is the easiest way to interact with the labs.

```bash
# Verify your environment dependencies
make doctor

# List all available labs
make list

# Start a specific lab
make up LAB=lab-11-production-grade-blueprint

# Run a chaos-injected benchmark (kills services mid-run)
make bench LAB=lab-11-production-grade-blueprint chaos=true

# Stop and cleanup
make down LAB=lab-11-production-grade-blueprint
```

## ⌨️ Advanced CLI (chatlab.py)

For more granular control, use the `scripts/chatlab.py` orchestrator directly.

### Benchmarking
Run the fair-comparison suite across all labs:
```bash
python3 scripts/chatlab.py suite
```

Include Lab 11 in the suite:
```bash
python3 scripts/chatlab.py suite --include-blueprint
```

### Observability & Logs
```bash
# Show URLs for Grafana, Jaeger, and Chat UI
make observe LAB=lab-11-production-grade-blueprint

# Follow service logs
python3 scripts/chatlab.py logs lab-11-production-grade-blueprint --follow
```

### Manual Failure Injection (Chaos)
You can manually inject failures to test the system's resilience:
```bash
# Kill a specific service
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint kill message-service

# Inject network latency to Redis
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint delay redis --latency-ms 300

# Heal a failure
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint heal redis
```

## 🛠️ Validation Gates
Run local validation gates to ensure system integrity:
```bash
python3 scripts/chatlab.py validate --kind workloads
python3 scripts/chatlab.py validate --kind slos
```
