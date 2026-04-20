# ChatLab
### *A Progressive Systems Engineering Curriculum*

ChatLab is a distributed-systems learning repo built around one evolving product: a real-time chat system. Each lab introduces one dominant systems problem, benchmarks the trade-off with the same comparable workload, and leaves behind a runnable architecture you can inspect locally.

This repo is benchmarked on a developer machine. It is meant for fair architectural comparison and systems learning, not for unsupported claims about internet-scale production capacity.

### What Changed In This Iteration
- one standard benchmark suite centered on `comparison_standard`
- one repo-level control script: `python3 scripts/chatlab.py ...`
- one guided learning path for what to read, run, and observe
- one architecture comparison report that now includes complexity, cost, scalability, failure handling, and real-world mapping
- one shared observability layer with Prometheus, Grafana, and Docker Compose logs
- one failure-injection workflow for node kill and network delay experiments
- one new capstone lab: [Lab 11](./labs/lab-11-production-grade-blueprint/README.md)

### Guided Learning Path
Use the same loop for every lab:
1. Read the lab README.
2. Start the stack with `python3 scripts/chatlab.py up <lab-name>`.
3. Open the observability endpoints with `python3 scripts/chatlab.py observe <lab-name>`.
4. Run the fair-comparison benchmark with `python3 scripts/chatlab.py bench <lab-name> --scenario comparison_standard`.
5. Compare the result against [results/comparison.md](./results/comparison.md).
6. Shut the stack down with `python3 scripts/chatlab.py down <lab-name>`.

The full guided sequence is in [docs/guided-learning-path.md](./docs/guided-learning-path.md).

### Standard Commands

```bash
python3 scripts/chatlab.py list
python3 scripts/chatlab.py up lab-01-monolith-baseline
python3 scripts/chatlab.py observe lab-01-monolith-baseline
python3 scripts/chatlab.py bench lab-01-monolith-baseline --scenario comparison_standard
python3 scripts/chatlab.py suite --scenario comparison_standard
python3 scripts/chatlab.py report
python3 scripts/chatlab.py down lab-01-monolith-baseline
```

The legacy interactive launcher in `main.py` still works, but `scripts/chatlab.py` is now the standard entrypoint.

### Standard Benchmark Suite

The fair-comparison suite is centered on one shared workload:
- scenario: `comparison_standard`
- load shape: 100 virtual users for 1 minute
- pacing: one message every 1000 ms
- payload target: 256 bytes
- summary outputs: latency, throughput, and error rate

The suite definition is documented in [docs/benchmark-suite.md](./docs/benchmark-suite.md).

### Repository Map
| Lab | Core concept | Problem focus | Real-world mapping | README |
| --- | --- | --- | --- | --- |
| Lab 01 | Monolith baseline | Find the in-memory latency floor | Prototype or hackathon MVP | [Lab 01](./labs/lab-01-monolith-baseline/README.md) |
| Lab 02 | Persistence layer | Make chat history durable | Small-team durable chat backend | [Lab 02](./labs/lab-02-persistence-layer/README.md) |
| Lab 03 | Redis pub/sub | Scale fan-out across nodes | Early WhatsApp-style brokered fan-out | [Lab 03](./labs/lab-03-redis-pubsub/README.md) |
| Lab 04 | Scalable monolith | Add backpressure inside one service | Queue-protected monolith | [Lab 04](./labs/lab-04-scalable-monolith/README.md) |
| Lab 05 | Cloud-native ingest | Decouple ingest from slow work | Netflix-style async ingest pipeline | [Lab 05](./labs/lab-05-cloud-native-chat-infrastructure/README.md) |
| Lab 06 | Chaos and resilience | Survive dependency failure safely | Resilience-first service mesh pattern | [Lab 06](./labs/lab-06-chaos-and-resilience/README.md) |
| Lab 07 | Presence and delivery | Scale ephemeral realtime state | WhatsApp or Discord-style realtime edge | [Lab 07](./labs/lab-07-real-time-presence-and-delivery/README.md) |
| Lab 08 | Global multi-region | Keep local latency low across regions | Multi-region messaging backbone | [Lab 08](./labs/lab-08-global-multi-region/README.md) |
| Lab 09 | Message security | Add confidentiality and replay defense | Signal-style secure messaging concerns | [Lab 09](./labs/lab-09-message-security/README.md) |
| Lab 10 | Microservices migration | Split reads, writes, and failure domains | Large-team service-oriented platform | [Lab 10](./labs/lab-10-microservices-migration/README.md) |
| Lab 11 | Production-grade blueprint | Consolidate the best decisions into one deployable stack | Pragmatic production-ready team blueprint | [Lab 11](./labs/lab-11-production-grade-blueprint/README.md) |

### Learning Arc
| Phase | Labs | Dominant question |
| --- | --- | --- |
| Foundations | 01-02 | What is the latency floor, and what does durability cost? |
| Runtime scaling | 03-04 | How do we scale fan-out and make backpressure explicit? |
| Cloud-native reliability | 05-06 | How do we decouple work and survive failure safely? |
| Distributed realtime state | 07-09 | How do presence, geography, and security change the design? |
| Service decomposition | 10 | What does service isolation buy, and what does it cost? |
| Deployable blueprint | 11 | What does a balanced, production-oriented local reference stack look like? |

### Observability
Every lab now exposes:
- Prometheus metrics
- Docker Compose logs
- Grafana dashboards
- benchmark artifacts under `benchmark/results/<run_id>/`

Use `python3 scripts/chatlab.py observe <lab-name>` to print the relevant URLs for the running stack.

### Failure Injection
Failure handling is part of the learning path now, not a late-stage extra.

Examples:

```bash
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience kill chat-worker
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience delay redis --latency-ms 300 --jitter-ms 50
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience heal redis
```

The workflow and suggested drills are documented in [docs/failure-injection.md](./docs/failure-injection.md).

### Key Docs
- [docs/benchmark-suite.md](./docs/benchmark-suite.md)
- [docs/benchmark-contract.md](./docs/benchmark-contract.md)
- [docs/guided-learning-path.md](./docs/guided-learning-path.md)
- [docs/failure-injection.md](./docs/failure-injection.md)
- [docs/final-system-spec.md](./docs/final-system-spec.md)
- [results/comparison.md](./results/comparison.md)

---
[Get Started with Lab 01](./labs/lab-01-monolith-baseline/README.md)
