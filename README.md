# ChatLab
### *A Progressive Systems Engineering Curriculum*

ChatLab is a phase-based curriculum for learning production-grade systems engineering through the lens of a real-time chat system. 

The structure is intentionally progressive: each lab introduces a new architectural constraint and resolves it at a larger scale. We move from a simple monolith to a globally resilient, cloud-native infrastructure capable of handling millions of concurrent users.

---

## 🧪 Research Benchmark Suite
The project includes a centralized benchmark suite driven by workload manifests in `benchmark/workloads/`. This keeps experiments reproducible while allowing controlled scenario changes.

**Phase 0/1 Contract:**
- **Benchmark Driver**: Python orchestrator + k6 + Prometheus scrape loop.
- **Workload Profiles**: YAML manifests (`robust_steady`, `latency_probe`, `spike_recovery`).
- **Per-Run Artifacts**: `metadata.json`, `timeseries.csv`, `k6_summary.json` in `benchmark/results/raw/<run_id>/`.
- **Compatibility Output**: `results/<lab>_robust_report.csv` for existing visualizer flow.

```bash
python3 main.py
```

Direct execution:

```bash
python3 benchmark/orchestrator.py lab-04-scalable-monolith robust_steady
```

---

## 🗺️ The Laboratory Roadmap

### Phase 1: Foundations (0 - 1k Users)
1. **[Lab 01: Monolith Baseline](./labs/lab-01-monolith-baseline)**
   * Single-node, in-memory. Focus: Broadcast loops and state management.
2. **[Lab 02: Persistence Layer](./labs/lab-02-persistence-layer)**
   * PostgreSQL durability. Focus: Synchronous I/O vs. UI responsiveness.

### Phase 2: Scaling the Monolith (1k - 10k Users)
3. **[Lab 03: Redis Pub/Sub](./labs/lab-03-redis-pubsub)**
   * Multi-node fan-out. Focus: Horizontal scaling and cross-node message delivery.
4. **[Lab 04: Scalable Monolith](./labs/lab-04-scalable-monolith)**
   * Async Worker Pools. Focus: Decoupling heavy tasks from the WebSocket hot path.

### Phase 3: Cloud-Native & Resilience (10k - 100k Users)
5. **[Lab 05: Cloud-Native Infrastructure](./labs/lab-05-cloud-native-chat-infrastructure)**
   * Object Storage & Ingest Queues. Focus: "Fire and Forget" ingestion and tiered retention.
6. **[Lab 06: Chaos & Resilience](./labs/lab-06-chaos-and-resilience)**
   * Circuit Breakers & Idempotency. Focus: Engineering for failure and guaranteed delivery.

### Phase 4: Social Semantic (100k - 1M Users)
7. **[Lab 07: Real-Time Presence](./labs/lab-07-real-time-presence-and-delivery)**
   * Presence Sync & Typing Signals. Focus: High-frequency ephemeral events and consistency.
8. **[Lab 08: Global Distribution](./labs/lab-08-global-multi-region)**
9. **[Lab 09: Message Security](./labs/lab-09-message-security)**

### Phase 5: The Mesh (1M+ Users)
10. **[Lab 10: Microservices Migration](./labs/lab-10-microservices-migration)**

---

## 📈 Standardized Metrics
All labs are instrumented with Prometheus to ensure fair benchmarking:
- `chat_active_connections`: Current WebSocket client count.
- `chat_messages_total`: Throughput capacity.
- `chat_message_latency_ms`: Ingest-to-Broadcast latency.
- `chat_memory_bytes`: Memory efficiency per user.

---
## 🧭 Distributed Behavior Contract

The advanced labs explicitly follow this contract so system behavior is predictable under load and failure.

### Consistency model
- **Global chat timeline**: eventual consistency across regions and services.
- **Per-room ordering**: best effort by timestamp and event id, not strict global total order.

### Delivery semantics
- **Inter-service replication**: at-least-once.
- **Client-visible delivery**: effectively-once where idempotency keys are enforced, otherwise at-least-once.

### Duplicate and reordering policy
- Duplicates are expected in distributed fan-out and retry paths.
- Every advanced lab uses message identifiers and deduplication maps/sets where applicable.
- Out-of-order arrival is handled at consumers with timestamp/event-id merge logic.

### Failure scenarios to validate
- Region outage during active message flow.
- Inter-region partition with delayed reconciliation.
- Clock skew between nodes.
- Partial replication and lagging region catch-up.

### Data ownership model
- Default strategy in global labs: **home-region ownership for writes** plus selective replication for read experience.
- Fully global replication is treated as an explicit tradeoff due to bandwidth and cost.

### Routing policy evolution
- Start: nearest-region routing.
- Then: sticky affinity and failover.
- Then: latency-aware and load-aware regional shifts.

### Cost awareness
- Separate mandatory global data from regional-only data.
- Track cross-region replication volume and archive bandwidth.
- Favor regional reads when global reads are not required.

---
[Get Started with Lab 01](./labs/lab-01-monolith-baseline/README.md)
