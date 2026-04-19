[🏠 Home](../../README.md) | [⬅️ Previous (Lab 01)](../lab-01-monolith-baseline/README.md) | [Next Lab (Lab 03) ➡️](../lab-03-redis-pubsub/README.md)

# Lab 02: The Persistence Layer
## *Durable State, SQL Overhead, and the Persistence Tax*

### 🔴 The Problem
In Lab 01, our chat was "Volatile." If the server crashed or restarted, every message was lost forever. For a real-world application, data must be **Durable**. However, writing to a disk-backed database (Postgres) is significantly slower than writing to RAM.
- **The Bottleneck**: Every message now requires a network round-trip to the database and a synchronous disk write.

### 🟢 The Approach
We introduce **PostgreSQL** to the architecture. Every incoming message is now persisted to a `messages` table before being broadcast. This lab allows us to measure the **"Persistence Tax"**—the exact latency penalty incurred by moving from in-memory state to a durable database.

---

### 🏗️ Architecture
![Lab 02 Architecture](assets/benchmarks/architecture.png)
*Figure 1: Architectural view of the Persistent Monolith with PostgreSQL.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Unified view of Latency, Load, Throughput, and Resource Utilization.*

#### 🧐 Analysis:
1. **The Persistence Tax**: Notice that the median latency has shifted from sub-1ms (Lab 01) to **~15ms**. This is the cost of durability.
2. **The Scaling Profile**: 
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Median latency response isolating the impact of SQL writes on system speed.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing the gap between expected and actual processing.*

#### 🧐 Analysis:
- **Throughput Deficit**: As load increases, the database becomes the primary bottleneck. The red area shows where the server begins to fall behind because it is waiting for disk I/O on every single message.

---

### 🔬 Key Lessons
- **Durability isn't Free**: The move to PostgreSQL introduces a measurable latency increase.
- **The SQL Bottleneck**: Moving state to a database solves the "Restart Data Loss" problem but creates a new scaling limit centered on Database I/O.

---

### 🚀 Commands
**Start the Lab:**
```bash
cd labs/lab-02-persistence-layer
docker-compose up --build -d
```

**Run Automated Benchmark:**
```bash
python3 labs/lab-02-persistence-layer/benchmark/run.py
```

**Generate Modern Graphs:**
```bash
python3 labs/lab-02-persistence-layer/benchmark/plot.py
```

---

### 📂 Folder Structure
- `services/chat-server/`: Go server with SQL integration.
- `benchmark/`: Automated orchestrator and analytics.
  - `run.py`: The automated benchmark orchestrator.
  - `plot.py`: The GitHub-Modern visualization engine.
- `assets/benchmarks/`: Permanent storage for persistence analytics.

---
[Next Lab: Lab 03 (Redis Pub/Sub) ➡️](../lab-03-redis-pubsub/README.md)
