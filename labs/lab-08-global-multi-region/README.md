[🏠 Home](../../README.md) | [⬅️ Previous (Lab 07)](../lab-07-real-time-presence-and-delivery/README.md)

# Lab 08: Global Multi-Region Distribution
## *Regional Isolation, Cross-Continental Bridges, and Event Deduplication*

Lab 08 explores the "Global" scale of chat systems. We move from a single cluster to a multi-region deployment spanning the **US** and **EU**. The objective is to achieve low local latency while maintaining global message consistency.

---

## 🏗️ Architecture

```
         🌍 US REGION                        🌉 GLOBAL BRIDGE                    🇪🇺 EU REGION
    ┌────────────────────┐              ┌────────────────────┐              ┌────────────────────┐
    │   Chat Node US     │◄────────────►│   Region Bridge    │◄────────────►│   Chat Node EU     │
    │ (Local Ingest)     │              │ (Stream Sync)      │              │ (Local Ingest)     │
    └────────┬───────────┘              └────────────────────┘              └────────┬───────────┘
             │                                                                       │
    ┌────────▼───────────┐                                                  ┌────────▼───────────┐
    │   Redis US         │                                                  │   Redis EU         │
    │ (Regional Stream)  │                                                  │ (Regional Stream)  │
    └────────────────────┘                                                  └────────────────────┘
```

---

## 📊 The Global Challenge

1. **Regional Latency Isolation**: Users in the US should only talk to US servers to avoid "Atlantic Round-Trip" latency (~100ms+) for local ingestion.
2. **Cross-Region Bridge**: A specialized service (The Bridge) monitors the US stream and replicates events to the EU stream (and vice versa).
3. **Event Deduplication**: To prevent infinite loops (US ➡️ Bridge ➡️ EU ➡️ Bridge ➡️ US), each node uses a `seenEvents` bloom-filter/map to drop messages it has already processed.

### Robust Benchmark Focus
In **Robust Mode**, we measure the **Synchronization Overhead**.
- **Local Latency**: We verify that US users still get sub-10ms ingest times.
- **Deduplication Efficiency**: We monitor the `chat_global_duplicate_events_total` metric to ensure the bridge isn't creating redundant traffic.

---

## 🔗 Endpoints
- **Chat UI (US Region)**: [http://localhost:8090](http://localhost:8090)
- **Chat UI (EU Region)**: [http://localhost:8091](http://localhost:8091)
- **Global Bridge Status**: [http://localhost:8092/status](http://localhost:8092/status)
- **Prometheus (Global)**: [http://localhost:9096](http://localhost:9096)

---

## 🚀 Run the Lab

```bash
cd labs/lab-08-global-multi-region
docker-compose up --build -d
```

## 🧪 Robust Benchmark
```bash
python3 main.py
```

---
[Next Lab: Lab 09 (Security & Encryption) ➡️](../lab-09-message-security/README.md)
