[🏠 Home](../../README.md) | [⬅️ Previous (Lab 10)](../lab-10-microservices-migration/README.md)

# Lab 11: Production-Grade Blueprint
## *Deployable Synthesis, Shared Controls, and Operational Readiness*

**Purpose:** consolidate the strongest decisions from the earlier labs into one deployable local blueprint with durable storage, gateway routing, metrics, dashboards, and standardized operational controls.  
**Hypothesis:** the best end-state is not the lowest-latency lab, but the most balanced system across latency, durability, observability, resilience, and operational clarity.

### Objective
This lab turns the curriculum into a runnable reference system. The goal is to end the series with a stack that a team could use as a serious starting point: gateway, persistence, broker, service separation, rate limiting, Prometheus, Grafana, and one standardized control script.

### Design Snapshot

| Lens | Answer |
| --- | --- |
| Problem | Earlier labs each solved one issue, but there was no single deployable stack that consolidated the best trade-offs. |
| Limitation | The repo could be read as a progression without a concrete operational finish line. |
| Solution | Package a capstone blueprint around gateway, message service, history service, Redis, Postgres, Prometheus, Grafana, and shared repo-level tooling. |
| Trade-off | This is the most operationally complete stack in the repo, but also the most expensive and complex to run. |

### What This Blueprint Includes
- gateway-mediated routing
- durable message storage
- Redis-backed event distribution
- standardized `scripts/chatlab.py` setup and benchmark commands
- Prometheus metrics and Grafana dashboards
- compatibility with the shared fair-comparison benchmark

### Standard Commands

```bash
python3 scripts/chatlab.py up lab-11-production-grade-blueprint
python3 scripts/chatlab.py observe lab-11-production-grade-blueprint
python3 scripts/chatlab.py bench lab-11-production-grade-blueprint --scenario comparison_standard
python3 scripts/chatlab.py down lab-11-production-grade-blueprint
```

### Suggested Failure Drills

```bash
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint kill history-service
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint delay redis --latency-ms 300 --jitter-ms 50
python3 scripts/chatlab.py fail lab-11-production-grade-blueprint heal redis
```

### Why This Is The Final Lab
Lab 11 is the capstone because it is no longer asking "what if we add X?" It asks the more realistic question: "Which combination of decisions gives us a system we can actually operate, observe, and evolve?"

### Real-World Mapping
This lab is intentionally not modeled after one named company. It is closer to the kind of pragmatic blueprint a product team would assemble after learning from systems like WhatsApp, Netflix, Signal, and modern service-oriented platforms.

### Commands

```bash
python3 scripts/chatlab.py up lab-11-production-grade-blueprint
python3 scripts/chatlab.py logs lab-11-production-grade-blueprint --follow
python3 scripts/chatlab.py report
```

---
[🏠 Return to Project Home](../../README.md)
