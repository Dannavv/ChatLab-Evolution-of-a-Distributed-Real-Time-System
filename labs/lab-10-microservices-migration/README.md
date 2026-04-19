[🏠 Home](../../README.md) | [⬅️ Previous (Lab 09)](../lab-09-message-security/README.md)

# Lab 10: Microservices Migration
## *Service Isolation, API Gateway Routing, and Distributed Observability*

Lab 10 represents the pinnacle of the architecture: a fully decoupled **Microservices Mesh**. We move from a monolithic secure node to a system where ingress, writes, and reads are handled by independent services.

---

## 🏗️ Architecture

```
                                  ┌───────────────────┐
                                  │   WebSocket Client│
                                  └─────────┬─────────┘
                                            │
        ┌──────────────┐          ┌─────────┴─────────┐          ┌──────────────┐
        │ History Svc  │◄────────┤    API Gateway    ├─────────►│ Message Svc  │
        │ (Read Path)  │          │ (Routing/Auth)    │          │ (Write Path) │
        └──────────────┘          └─────────┬─────────┘          └──────────────┘
                                            │
                                  ┌─────────┴─────────┐
                                  │   Redis / Postgres│
                                  └───────────────────┘
```

---

## 📊 Performance Analysis (The "Microservice Cliff")
![Lab 10 Performance](../../assets/benchmarks/lab-10-microservices-migration-performance.png)

### The Real-World "Microservice Tax"
The **Robust Stress Test** for Lab 10 reveals the most critical lesson in the entire curriculum: **Decoupling is not free.**

1. **Cascading Latency (23s+)**: At the 2,500 VU peak, latency spiked to an astronomical **23 seconds**. This occurred because the Gateway and Message Service exhausted their 0.5 CPU limits just managing the HTTP/TCP overhead of communicating with each other. This is the "Microservice Tax" in its most extreme form.
2. **Resource Exhaustion**: Notice the memory usage climbed to **~230MB** across the mesh. While each individual service is lean, the aggregate overhead of multiple Go runtimes and HTTP buffers significantly exceeds the footprint of the Lab 05 Cloud-Native monolith.
3. **The Throughput Wall**: Unlike previous labs that could handle the 2,500 VU peak with ~100ms latency, the Lab 10 mesh effectively "stalled" at 1,000 VUs. This demonstrates that for small-scale deployments, a monolith is often *faster* than a microservice mesh.

---

## 🔬 Service Breakdown
- **API Gateway (8100)**: The single entry point. Handles WebSocket connections, enforces rate limits, and proxies requests to backend services.
- **Message Service (8101)**: Dedicated to the "Write" path. Saves to PostgreSQL and publishes to the Redis event bus.
- **History Service (8102)**: Dedicated to the "Read" path. Serves historical room messages from PostgreSQL.
- **Unified Observability**: Prometheus aggregates metrics from all three services on port 9100.

---

## 🔗 Endpoints
- **Chat UI (via Gateway)**: [http://localhost:8100](http://localhost:8100)
- **Gateway Status**: [http://localhost:8100/status](http://localhost:8100/status)
- **Message Svc Metrics**: [http://localhost:8101/metrics](http://localhost:8101/metrics)
- **Prometheus (Mesh)**: [http://localhost:9100](http://localhost:9100)

---

## 🚀 Run the Mesh

```bash
cd labs/lab-10-microservices-migration
docker-compose up --build -d
```

## 🧪 Robust Benchmark
```bash
python3 main.py
```

---
[Next Lab: Lab 11 (Serverless Integration) ➡️](../lab-11-serverless-functions/README.md)
