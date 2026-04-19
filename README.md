# ChatLab
### *A Progressive Systems Engineering Curriculum*

ChatLab is a progressive distributed-systems curriculum built around one evolving product: a real-time chat system. Each lab keeps the previous lab's core behavior in view, introduces one new production constraint, and then measures the architectural trade-offs with a local benchmark harness.

### 🎯 Project Objective
The goal of the repository is to teach systems design through concrete, runnable deltas rather than isolated theory. We start with a single-process baseline, then add durability, distribution, queueing, resilience, presence, multi-region routing, security, and finally service decomposition.

### 🧭 How To Use This Repo
1. Start with [Lab 01](./labs/lab-01-monolith-baseline/README.md) and move in order.
2. Use the lab README to understand the architectural problem each stage solves.
3. Use the benchmark README inside each lab to run and inspect the local workload harness.
4. Use `python3 main.py` if you want a simple benchmark launcher for all labs that expose `benchmark/run.py`.

### 🚀 Benchmark Orchestrator
`main.py` is the repo-level benchmark entry point. It scans `labs/` for benchmark runners and lets you execute one lab or the full available suite.

```bash
python3 main.py
```

### 📚 Repository Map
| Lab | Theme | Lab README | Benchmark README |
| --- | --- | --- | --- |
| Lab 01 | Monolith baseline | [Lab 01](./labs/lab-01-monolith-baseline/README.md) | [Benchmark](./labs/lab-01-monolith-baseline/benchmark/README.md) |
| Lab 02 | Persistence layer | [Lab 02](./labs/lab-02-persistence-layer/README.md) | [Benchmark](./labs/lab-02-persistence-layer/benchmark/README.md) |
| Lab 03 | Redis pub/sub | [Lab 03](./labs/lab-03-redis-pubsub/README.md) | [Benchmark](./labs/lab-03-redis-pubsub/benchmark/README.md) |
| Lab 04 | Scalable monolith | [Lab 04](./labs/lab-04-scalable-monolith/README.md) | [Benchmark](./labs/lab-04-scalable-monolith/benchmark/README.md) |
| Lab 05 | Cloud-native ingest | [Lab 05](./labs/lab-05-cloud-native-chat-infrastructure/README.md) | [Benchmark](./labs/lab-05-cloud-native-chat-infrastructure/benchmark/README.md) |
| Lab 06 | Chaos and resilience | [Lab 06](./labs/lab-06-chaos-and-resilience/README.md) | [Benchmark](./labs/lab-06-chaos-and-resilience/benchmark/README.md) |
| Lab 07 | Presence and delivery | [Lab 07](./labs/lab-07-real-time-presence-and-delivery/README.md) | [Benchmark](./labs/lab-07-real-time-presence-and-delivery/benchmark/README.md) |
| Lab 08 | Global multi-region | [Lab 08](./labs/lab-08-global-multi-region/README.md) | [Benchmark](./labs/lab-08-global-multi-region/benchmark/README.md) |
| Lab 09 | Message security | [Lab 09](./labs/lab-09-message-security/README.md) | [Benchmark](./labs/lab-09-message-security/benchmark/README.md) |
| Lab 10 | Microservices migration | [Lab 10](./labs/lab-10-microservices-migration/README.md) | [Benchmark](./labs/lab-10-microservices-migration/benchmark/README.md) |

### 🗺️ Roadmap
#### Phase 1: Foundations
- **Lab 01** defines the single-node in-memory latency floor and the limits of local state.
- **Lab 02** adds durable storage and measures the persistence tax.

#### Phase 2: Scaling The Runtime
- **Lab 03** moves fan-out onto Redis so multiple chat nodes can share one bus.
- **Lab 04** protects a single node with internal queueing and worker pools.

#### Phase 3: Cloud-Native Reliability
- **Lab 05** separates ingest from processing and adds long-lived storage concerns.
- **Lab 06** adds circuit breaking, retries, and dead-letter handling.

#### Phase 4: High-Frequency State And Geography
- **Lab 07** models presence, typing, and delivery signals as their own scaling problem.
- **Lab 08** introduces regional isolation and asynchronous inter-region bridges.
- **Lab 09** layers in encryption and key-management overhead.

#### Phase 5: Service Decomposition
- **Lab 10** breaks the final single service boundary into a gateway plus focused backend services.

### 📈 Benchmark Contract
Across the labs, the benchmark harness follows the same broad pattern:
- `benchmark/workload.yaml` defines the scenario shape.
- `benchmark/run.py` starts the stack, waits for health, samples Prometheus, runs the load driver, and writes per-run artifacts.
- `benchmark/plot.py` turns `timeseries.csv` into run and suite graphs.
- `benchmark/results/<run_id>/` stores the artifacts for each executed scenario.

Outputs vary slightly by lab generation, but the common artifacts are:
- `metadata.json`
- `timeseries.csv`
- `graphs/*.png`

Labs 01 and 02 additionally export richer benchmark summaries because their harnesses were upgraded earlier in the curriculum.

### 📏 Shared Metrics
These metrics appear repeatedly across the README set:
- `latency`: end-to-end time from client send to observed response, or the closest available server-side proxy in older harnesses.
- `throughput`: messages processed per second.
- `error rate`: dropped, failed, or diverted work relative to processed traffic.
- `db latency`: database time measured separately in labs that expose a persistence path.
- `active connections`: concurrent WebSocket sessions under load.

### 🧠 Reading The Labs
Each lab README now answers the same questions:
- What problem does this lab solve?
- What changed from the previous lab?
- What does the request path look like now?
- Why did performance improve or degrade?
- What limitation remains, and what does the next lab unlock?

### ✅ Suggested Starting Point
- Read [Lab 01](./labs/lab-01-monolith-baseline/README.md)
- Run [Lab 01 Benchmark](./labs/lab-01-monolith-baseline/benchmark/README.md)
- Move sequentially through the roadmap

---
[Get Started with Lab 01](./labs/lab-01-monolith-baseline/README.md)
