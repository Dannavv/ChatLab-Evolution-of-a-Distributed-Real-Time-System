# ChatLab
### *A Progressive Systems Engineering Curriculum*

ChatLab is a progressive distributed-systems curriculum built around one evolving product: a real-time chat system. Each lab keeps the previous lab's core behavior in view, introduces one dominant systems constraint, and measures the trade-off with a shared local benchmark harness.

The repo is intentionally benchmarked on a local developer machine, not positioned as an internet-scale production reference. What it does provide is a reproducible path from single-process chat to a service-oriented distributed system, with the performance, reliability, consistency, routing, failure, observability, and cost implications made explicit at each step.

### Project Objective
The goal of the repository is to teach systems design through runnable deltas rather than isolated theory.

We start with a single-process WebSocket server, then progressively introduce:
- durability
- brokered fan-out
- backpressure
- queue-based decomposition
- resilience controls
- presence and delivery state
- multi-region routing
- message security
- microservice boundaries

The curriculum is successful when we can answer three questions for every lab:
- What single architectural problem does this lab solve?
- What measurable trade-off did we introduce?
- Why is the next lab necessary?

### Evidence, Not Hype
The current repo-wide comparison comes from the latest `comparison_standard` runs recorded on April 19, 2026. These numbers are workstation-local and should be read as comparative signals, not universal capacity claims.

| Signal | Current observed range |
| --- | --- |
| P95 latency | `1.26 ms` to `9.05 ms` |
| Average throughput | `1.83` to `67.22 msgs/s` |
| Highest observed error rate | `29.52%` in Lab 04 under queue saturation |
| Lowest-latency comparable lab | Lab 07 at `1.26 ms` p95 |
| Highest-throughput comparable lab | Lab 03 at `67.22 msgs/s` |

Source: [results/comparison.md](./results/comparison.md)

### How To Use This Repo
1. Start with [Lab 01](./labs/lab-01-monolith-baseline/README.md) and move in order.
2. Read the lab objective first; each lab is intended to teach one dominant concept.
3. Run the benchmark for that lab before moving on.
4. Compare the lab's results against the repo-wide report.
5. Use [docs/final-system-spec.md](./docs/final-system-spec.md) as the capstone reference for how the pieces fit together.

### Benchmark Orchestrator
`main.py` is the repo-level benchmark entry point. It scans `labs/` for benchmark runners and lets you execute one lab or the full available suite.

```bash
python3 main.py
```

The unified comparison report is written to [results/comparison.md](./results/comparison.md).

### Repository Map
| Lab | Core concept | Primary question | Lab README | Benchmark README |
| --- | --- | --- | --- | --- |
| Lab 01 | Monolith baseline | How fast is the in-memory latency floor? | [Lab 01](./labs/lab-01-monolith-baseline/README.md) | [Benchmark](./labs/lab-01-monolith-baseline/benchmark/README.md) |
| Lab 02 | Persistence layer | What does synchronous durability cost? | [Lab 02](./labs/lab-02-persistence-layer/README.md) | [Benchmark](./labs/lab-02-persistence-layer/benchmark/README.md) |
| Lab 03 | Redis pub/sub | How do we scale fan-out across nodes? | [Lab 03](./labs/lab-03-redis-pubsub/README.md) | [Benchmark](./labs/lab-03-redis-pubsub/benchmark/README.md) |
| Lab 04 | Scalable monolith | What does explicit backpressure look like inside one service? | [Lab 04](./labs/lab-04-scalable-monolith/README.md) | [Benchmark](./labs/lab-04-scalable-monolith/benchmark/README.md) |
| Lab 05 | Cloud-native ingest | How do we separate ingest from slow processing? | [Lab 05](./labs/lab-05-cloud-native-chat-infrastructure/README.md) | [Benchmark](./labs/lab-05-cloud-native-chat-infrastructure/benchmark/README.md) |
| Lab 06 | Chaos and resilience | How does the pipeline behave under dependency failure? | [Lab 06](./labs/lab-06-chaos-and-resilience/README.md) | [Benchmark](./labs/lab-06-chaos-and-resilience/benchmark/README.md) |
| Lab 07 | Presence and delivery | How do ephemeral real-time signals scale differently from messages? | [Lab 07](./labs/lab-07-real-time-presence-and-delivery/README.md) | [Benchmark](./labs/lab-07-real-time-presence-and-delivery/benchmark/README.md) |
| Lab 08 | Global multi-region | What consistency and routing trade-offs appear across regions? | [Lab 08](./labs/lab-08-global-multi-region/README.md) | [Benchmark](./labs/lab-08-global-multi-region/benchmark/README.md) |
| Lab 09 | Message security | What is the measurable cost of confidentiality and key rotation? | [Lab 09](./labs/lab-09-message-security/README.md) | [Benchmark](./labs/lab-09-message-security/benchmark/README.md) |
| Lab 10 | Microservices migration | What new latency and operational overhead come with service boundaries? | [Lab 10](./labs/lab-10-microservices-migration/README.md) | [Benchmark](./labs/lab-10-microservices-migration/benchmark/README.md) |

### Lab Progression
| Phase | Labs | Dominant objective |
| --- | --- | --- |
| Foundations | 01-02 | Establish the latency floor, then make durability measurable |
| Runtime scaling | 03-04 | Scale fan-out and introduce explicit backpressure |
| Cloud-native reliability | 05-06 | Decouple ingest from processing and survive dependency failure |
| Real-time distributed state | 07-09 | Model presence, geography, and security as first-class concerns |
| Service decomposition | 10 | Split responsibilities into independently operated services |

### Real-Time SLO Lens
The repo focuses on real-time behavior, so every lab should be read through the same operational lens:
- `latency`: track p50, p95, and p99 rather than averages alone
- `throughput`: messages successfully processed per second
- `delivery ratio`: messages received relative to sent
- `error rate`: failed, dropped, or diverted work relative to processed traffic
- `jitter`: variance between median and tail latency during steady load
- `deadline thinking`: whether the system can meet a user-visible target such as "new message visible within 100 ms"

A useful way to extend the labs is to assign explicit SLOs such as:
- p95 end-to-end message latency `< 50 ms` for single-region interactive traffic
- delivery ratio `>= 99.9%` for accepted messages
- error budget `< 0.1%` per benchmark run

### Benchmark Methodology
Across the labs, the benchmark harness follows one repo-wide contract:
- `benchmark/workload.yaml` defines the workload shape
- `benchmark/run.py` delegates to the shared runner in `shared/benchmark/framework.py`
- `benchmark/plot.py` delegates to `shared/benchmark/plotting.py`
- `benchmark/results/<run_id>/` stores artifacts for each executed scenario
- `shared/benchmark/report.py` synthesizes the latest comparable run from each lab into [results/comparison.md](./results/comparison.md)

Common artifacts are:
- `metadata.json`
- `timeseries.csv`
- `benchmark_summary.json`
- `graphs/*.png`

The contract is documented in [docs/benchmark-contract.md](./docs/benchmark-contract.md).

### Observability Baseline
Observability should not start late in the series. The repo already standardizes Prometheus metrics and benchmark artifacts, and later labs add more detailed telemetry. A strong extension path is:
- Prometheus metrics from Lab 01 onward
- trace propagation using `trace_id` and `message_id` across every hop
- OpenTelemetry spans for ingress, broker handoff, persistence, and fan-out
- latency distribution analysis from `timeseries.csv`, not only headline averages
- per-lab dashboards for throughput, queue depth, retry volume, and failure modes

### Failure and Resilience Model
Failure handling is part of the curriculum, not an afterthought. The progression currently includes overload, database stalls, broker issues, queue saturation, retries, dead-letter routing, stale presence, and cross-region lag.

A stronger extension path for future work is to add deliberate failure injection:
- kill a chat node during steady traffic
- delay broker or database responses
- introduce packet loss and network jitter
- partition one region from another
- verify recovery time, backlog replay behavior, and data loss boundaries

### Production Readiness Gaps
This repo is a learning system with real measurements, not a finished production platform. Important gaps remain intentionally visible:
- authentication and authorization flows
- RBAC for admin and operational actions
- rate limiting and abuse controls
- secret rotation and secure configuration delivery
- threat modeling and OWASP-style web hardening
- reproducible multi-stage test pipelines across unit, integration, load, chaos, and security testing

Lab 09 and Lab 10 are natural places to deepen these topics.

### System Design Depth To Watch For
As the labs progress, the important trade-offs are no longer just "faster" or "slower." They become:
- immediate visibility vs eventual consistency
- low latency vs durable writes
- direct fan-out vs broker dependence
- queue absorption vs backlog growth
- regional isolation vs cross-region coherence
- stronger security vs cryptographic overhead
- simple deployments vs service-boundary complexity

These are the same kinds of trade-offs that show up in CAP discussions, queue design, partitioning strategy, and consistency-model selection in real systems.

### Advanced Patterns To Add Next
The current ten-lab arc covers a strong core, but a next wave of advanced topics would fit naturally after the existing sequence:
- event sourcing for immutable message history and replay
- CQRS for separating write-path durability from read-optimized delivery views
- durable broker semantics and explicit backpressure thresholds
- partition-aware sharding for room or tenant isolation
- replica lag analysis and read-after-write consistency tests
- cost modeling that ties infrastructure choices to throughput and latency gains

### Final System Spec
The series now ends with an explicit capstone target in [docs/final-system-spec.md](./docs/final-system-spec.md). That document defines:
- the final service topology
- core data flows
- proposed SLOs and failure budgets
- security and testing expectations
- known trade-offs and open extension areas

### Suggested Starting Point
- Read [Lab 01](./labs/lab-01-monolith-baseline/README.md)
- Run [Lab 01 Benchmark](./labs/lab-01-monolith-baseline/benchmark/README.md)
- Read [docs/benchmark-contract.md](./docs/benchmark-contract.md)
- Read [docs/final-system-spec.md](./docs/final-system-spec.md)
- Move sequentially through the roadmap

---
[Get Started with Lab 01](./labs/lab-01-monolith-baseline/README.md)
