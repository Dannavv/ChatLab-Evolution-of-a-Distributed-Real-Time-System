# ChatLab: Evolution of a Distributed Real-Time System

![ChatLab Cover](./assets/cover.png)

ChatLab is a curated journey through the evolution of distributed systems. It transforms a simple stateful monolith into a **hardened, production-grade microservices blueprint** that handles global scale, partial failures, and deep observability.

## Hook
Build and benchmark the same chat product through 11 architecture stages, then use real p95/throughput/reliability data to decide which design is right for a production constraint, not just a textbook pattern.

## Learning Outcomes
- Explain how each architectural step changes latency, throughput, and reliability behavior.
- Evaluate backpressure, durability, consistency, and failure-handling trade-offs with benchmark evidence.
- Select an architecture based on product constraints, team boundaries, and operational risk.

## Why This Matters in Production
Most architecture guides show static diagrams; this repository shows measured trade-offs under load and failure. The goal is to help you make better decisions when budgets, incidents, and scaling pressure are real.

> [!IMPORTANT]
> **Hardened Edition:** This repository now includes production-grade resilience patterns, including Circuit Breakers, Global Rate Limiting, Distributed Tracing (Jaeger), and Automated Chaos Testing.

## 🛠️ Quick Start (The "System Owner" Way)

The primary entry point for all operations is the root `Makefile`.

Optional local environment bootstrap:

```bash
cp .env.example .env
```

```bash
# 1. Check your environment
make doctor

# 2. List all available labs
make list

# 3. Spin up the final capstone blueprint
make up LAB=lab-11-production-grade-blueprint

# 4. Run a chaos-injected benchmark (kills services mid-run)
make bench LAB=lab-11-production-grade-blueprint chaos=true
```

## 📖 The "Hardened" Architecture
This project isn't just about "working" code; it's about "operating" code. The final labs implement:

- **Resilience:** Circuit Breakers, Jittered Exponential Backoff, and Graceful Shutdown.
- **Observability:** Distributed Tracing with OpenTelemetry and Jaeger.
- **Security:** ULID stable ID generation and Idempotency hardening.
- **Rigor:** Automated failure injection (Chaos) with Recovery Time (RTO) metrics.

## 📊 System Evolution Summary

Quick comparison from the latest `comparison_standard` runs. Full details remain in `results/comparison.md`.

| Lab | Architecture | Avg Latency (p50) | P95 Latency | Throughput |
| :--- | :--- | :---: | :---: | :---: |
| 1 | Monolith baseline | 2.05 ms | 2.80 ms | 65.07 msgs/s |
| 2 | Monolith + PostgreSQL persistence | 2.76 ms | 3.50 ms | 65.04 msgs/s |
| 3 | Redis Pub/Sub distributed mesh | 1.30 ms | 1.55 ms | 67.22 msgs/s |
| 4 | Scalable monolith (worker pool) | 1.74 ms | 2.14 ms | 49.29 msgs/s |
| 5 | Cloud-native async pipeline | 1.40 ms | 1.62 ms | 65.04 msgs/s |
| 6 | Chaos + resilience controls | 1.43 ms | 1.62 ms | 65.04 msgs/s |
| 7 | Presence + delivery tracking | 1.08 ms | 1.26 ms | 65.04 msgs/s |
| 8 | Global multi-region bridge | 1.50 ms | 2.00 ms | 1.83 msgs/s |
| 9 | Message security hardening | 1.24 ms | 1.60 ms | 64.82 msgs/s |
| 10 | Microservices migration | 7.49 ms | 9.05 ms | 65.04 msgs/s |
| 11 | Production-grade blueprint | 7.91 ms | 9.11 ms | 43.52 msgs/s |

## ⌨️ CLI Command Reference

While you can use `scripts/chatlab.py` directly, the `Makefile` is the recommended interface to avoid pathing confusion.

| Command | Description |
| :--- | :--- |
| `make doctor` | **Run this first.** Checks if Docker, Go, k6, and Python are ready. |
| `make up LAB=<name>` | Starts the Docker stack for a specific lab. |
| `make down LAB=<name>` | Stops and cleans up the stack. |
| `make bench LAB=<name>` | Runs k6 benchmarks and generates a performance report. |
| `make bench ... chaos=true` | Injects a service failure mid-benchmark to test resilience. |
| `make status LAB=<name>` | Checks if all containers are healthy. |
| `make observe LAB=<name>` | Shows the URLs for the Chat UI, Grafana, and Jaeger. |
| `make suite` | Runs benchmarks for all labs sequentially to generate a comparison report. |
| `make report` | Manually regenerates the `results/COMPARISON.md` report. |

---

## 🎓 Curriculum Path

1. **[Lab 01: Monolith Baseline](./labs/lab-01-monolith-baseline)** - Single service, in-memory state.
2. **[Lab 02: Persistence Layer](./labs/lab-02-persistence-layer)** - Adding PostgreSQL for durability.
3. **[Lab 03: Redis Pub/Sub](./labs/lab-03-redis-pubsub)** - Scaling out the WebSocket tier.
4. **[Lab 04: Scalable Monolith](./labs/lab-04-scalable-monolith)** - Handling high-concurrency with Go routines.
5. **[Lab 05: Cloud-Native Infra](./labs/lab-05-cloud-native-chat-infrastructure)** - Metrics (Prometheus) and Dashboards (Grafana).
6. **[Lab 06: Chaos & Resilience](./labs/lab-06-chaos-and-resilience)** - Intro to retries and partial failures.
7. **[Lab 07: Presence & Delivery](./labs/lab-07-real-time-presence-and-delivery)** - Redis sets for user tracking.
8. **[Lab 08: Global Multi-Region](./labs/lab-08-global-multi-region)** - Geographic partitioning and latency.
9. **[Lab 09: Message Security](./labs/lab-09-message-security)** - HMAC signatures and encryption.
10. **[Lab 10: Microservices Migration](./labs/lab-10-microservices-migration)** - Splitting into Gateway, Message, and History services.
11. **[Lab 11: Production-Grade Blueprint](./labs/lab-11-production-grade-blueprint)** - **THE HARDENED CAPSTONE.**

---

## 📑 Executive Summary
The ChatLab “Evolution” repository shows a clear stepwise build-out of a chat system. This curriculum transitions from a simple stateful monolith to a global, multi-region distributed architecture. Each lab is hardened to introduce industry-standard patterns: durable message logs, CAP-theorem-aware design, stateless service architectures, and deep observability (Tracing, Metrics, Logs).

## 🚀 Standard Commands

If you prefer the raw Python CLI:

```bash
python3 scripts/chatlab.py list
python3 scripts/chatlab.py up lab-01-monolith-baseline
python3 scripts/chatlab.py observe lab-01-monolith-baseline
python3 scripts/chatlab.py bench lab-01-monolith-baseline --scenario comparison_standard
python3 scripts/chatlab.py suite --scenario comparison_standard
python3 scripts/chatlab.py report
python3 scripts/chatlab.py down lab-01-monolith-baseline
```
