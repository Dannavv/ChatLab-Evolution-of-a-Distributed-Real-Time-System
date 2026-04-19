[🏠 Home](../../README.md) | [⬅️ Previous (Lab 04)](../lab-04-scalable-monolith/README.md) | [Next Lab (Lab 06) ➡️](../lab-06-chaos-and-resilience/README.md)

# Lab 05: Cloud-Native Chat Infrastructure
## *Decoupled Pipelines and Object Storage*

### 🔬 The Hypothesis
> "By decoupling the ingest path (API) from the processing path (Worker) using a Redis Queue, we can achieve high 'Burst Tolerance.' The system will accept messages at wire-speed and process them asynchronously, allowing us to leverage scalable Object Storage (MinIO) for long-term archiving without affecting real-time latency."

### 🔴 The Problem: The Heavyweight Worker
In Lab 04, if a worker was slow, the whole server lagged. 
- **The Limit**: If you want to archive messages to S3/MinIO, the write time can be unpredictable. 
- **The Solution**: **Micro-Batching & Async Processing**. The API node only writes to a fast Redis Queue. A separate Worker node pulls from Redis and handles the heavy lifting (Postgres writes + MinIO archiving).

---

### 🏗️ Architecture
![Lab 05 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Cloud-Native Pipeline. API Gateway -> Redis Stream -> Background Worker -> Object Storage.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh showing the decoupled API response times.*

#### 🧐 Reading the Signal:
1.  **Ingest Speed**: Notice that "Latency" (API response) remains incredibly low even as the workload spikes. This is because the API is only doing a single Redis `LPUSH`.
2.  **The Decoupling Proof**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: API Latency vs Load. The flat line proves that the "Heavy" processing (DB/MinIO) has been successfully removed from the critical path.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit.*

#### 🧐 Reading the Signal:
- **Queue Buffering**: Unlike previous labs where "Deficit" meant "Dropped Data," in Lab 05, a deficit often just means the **Worker is behind**. The messages are safe in the Redis Queue and will be processed once the load subsides. This is **Durability by Design**.

---

### 🔬 Key Lessons
- **Critical Path Management**: Never do I/O (Disk/S3) in a WebSocket handler.
- **Object Storage vs. Relational**: PostgreSQL handles the "Real-Time History," while MinIO handles the "Permanent Archive."

---

### 🚀 Commands
```bash
# Start the full stack (API, Worker, Redis, DB, MinIO)
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-05-cloud-native-chat-infrastructure/benchmark/run.py
```

---
[Next Lab: Lab 06 (Chaos & Resilience) ➡️](../lab-06-chaos-and-resilience/README.md)
