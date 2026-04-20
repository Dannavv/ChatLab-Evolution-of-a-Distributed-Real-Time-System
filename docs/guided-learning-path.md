# ChatLab Guided Learning Path

This path is designed to keep the repo readable as one evolving system rather than ten disconnected demos.

## Read, Run, Observe Loop

For each lab, follow the same order:
1. Read the lab README and identify the single architectural problem being introduced.
2. Start the stack with `python3 scripts/chatlab.py up <lab-name>`.
3. Open the observability endpoints with `python3 scripts/chatlab.py observe <lab-name>`.
4. Run the fair comparison benchmark with `python3 scripts/chatlab.py bench <lab-name> --scenario comparison_standard`.
5. Compare the result against [results/comparison.md](../results/comparison.md).
6. Shut the lab down with `python3 scripts/chatlab.py down <lab-name>`.

## Lab By Lab

| Lab | Problem | Limitation | Solution | Trade-off | Real-world mapping |
| --- | --- | --- | --- | --- | --- |
| Lab 01 | Establish the absolute latency floor for chat | State is volatile and scaling stops at one node | Single-process in-memory websocket server | Lowest complexity, weakest durability | Early-stage MVP or internal prototype |
| Lab 02 | Keep messages after restart | Database writes add storage cost and latency | Add PostgreSQL-backed history | Durability improves, tail latency and ops burden rise | Slack-style history layer before horizontal fan-out |
| Lab 03 | Broadcast across multiple nodes | Pub/sub is fast but not strongly durable | Redis-based shared fan-out bus | Better horizontal scale, eventual consistency | Early WhatsApp-style brokered fan-out layer |
| Lab 04 | Prevent one node from collapsing under burst load | Internal queues can still saturate and drop work | Add worker pools and explicit backpressure | Better burst handling, more queue tuning | A queue-protected monolith used before service split |
| Lab 05 | Keep ingest fast while heavy work happens later | Async pipelines can hide backlog until it becomes dangerous | Decouple API from worker and archive paths | Lower ingest latency, eventual completion semantics | Netflix-style async media/control pipeline pattern |
| Lab 06 | Survive downstream failure without cascading collapse | Retries and DLQs add more moving parts | Add circuit breakers, retries, and dead-letter flow | Safer failure behavior, higher operational complexity | Resilience patterns common in Netflix/Hystrix-era systems |
| Lab 07 | Handle ephemeral presence and delivery state separately from durable messages | Freshness is hard to guarantee at scale | Use stateful websocket routing and soft-state coordination | Presence stays fast, consistency becomes intentionally soft | WhatsApp/Discord-style presence-heavy realtime edge |
| Lab 08 | Keep local traffic fast across regions | Global convergence is no longer immediate | Add regional affinity and async replication bridges | Better local UX, harder cross-region consistency story | Multi-region chat or social messaging backbone |
| Lab 09 | Protect confidentiality and integrity | Crypto burns CPU and rotation adds jitter | Add encryption, validation, and replay defense | Better security, lower raw throughput headroom | Signal-style secure messaging concerns layered onto distributed chat |
| Lab 10 | Isolate reads, writes, and failure domains | More hops mean more baseline latency and more observability work | Split gateway, message, and history services | Better organizational scaling, higher platform cost | Service-oriented systems used by large platform teams |
| Lab 11 | Consolidate the best operational decisions into one deployable blueprint | Production readiness needs explicit SLOs, auth, observability, and resilience policy | Compose the capstone system with gateway, durable storage, broker, metrics, and dashboards | Most capable design, highest implementation and ops cost | A pragmatic production-ready team blueprint rather than a single company clone |

## What To Observe

Use the same questions every time:
- Did p95 latency improve or worsen?
- What became the dominant bottleneck?
- Which failure mode is now explicit instead of hidden?
- What new operational cost was introduced?
- Why does the next lab exist?
