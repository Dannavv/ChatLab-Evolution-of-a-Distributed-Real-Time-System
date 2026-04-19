# ChatLab
### *A Progressive Systems Engineering Curriculum*

ChatLab is a phase-based curriculum for learning production-grade systems engineering through the lens of a real-time chat system. 

The structure is intentionally progressive: each lab introduces a new architectural constraint and resolves it at a larger scale. We move from a simple monolith to a globally resilient, cloud-native infrastructure capable of handling millions of concurrent users.

---

## 🧪 Robust Performance Suite
The project includes a centralized **Robust Stress Test** suite. This suite performs a standardized, high-pressure evaluation of every lab architecture to identify its "Efficiency Cliff."

**Suite Parameters:**
- **Hardware Limit**: 0.5 CPU / 512MB RAM per node.
- **Max Load**: 2,500 Virtual Users (VUs).
- **Duration**: 4.0 Minutes (Accelerated Ramp).
- **Tooling**: Python Orchestrator + k6 + Prometheus.

```bash
python3 main.py
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
8. **Lab 08: Global Distribution** (Coming Soon)
9. **[Lab 09: Message Security](./labs/lab-09-message-security)**

### Phase 5: The Mesh (1M+ Users)
10. **Lab 10: Microservices Migration** (Coming Soon)

---

## 📈 Standardized Metrics
All labs are instrumented with Prometheus to ensure fair benchmarking:
- `chat_active_connections`: Current WebSocket client count.
- `chat_messages_total`: Throughput capacity.
- `chat_message_latency_ms`: Ingest-to-Broadcast latency.
- `chat_memory_bytes`: Memory efficiency per user.

---
[Get Started with Lab 01](./labs/lab-01-monolith-baseline/README.md)
