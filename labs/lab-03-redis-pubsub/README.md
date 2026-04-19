[🏠 Home](../../README.md) | [⬅️ Previous (Lab 02)](../lab-02-persistence-layer/README.md) | [Next Lab (Lab 04) ➡️](../lab-04-scalable-monolith/README.md)

# Lab 03: Redis Pub/Sub
## *The Distributed Mesh and Linear Scaling*

### 🔬 The Hypothesis
> "By decoupling the message distribution from the application server using Redis Pub/Sub, we can achieve linear horizontal scaling. Multiple server nodes can now share a unified message bus, allowing us to distribute client connections without losing global chat connectivity."

### 🔴 The Problem: The Single-Node Wall
In Lab 01 and 02, our "Broadcast Loop" was synchronous and local. 
- **The Limit**: If you added a second server, users on Server A couldn't talk to users on Server B.
- **The Solution**: A centralized **Message Broker (Redis)** that acts as a global backbone.

---

### 🏗️ Architecture
![Lab 03 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Distributed Architecture. Multiple chat nodes connected via Redis Pub/Sub.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Unified view of the Distributed Mesh performance.*

#### 🧐 Reading the Signal:
1.  **The Efficiency Breakthrough**: Unlike the Monolith (Lab 01), which hit a wall at 100 users, Lab 03 demonstrates **Stable Latency** even as we scale across multiple nodes.
2.  **Horizontal Proof**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Scaling Profile. Note how the latency curve is significantly flatter compared to Lab 02, as the work is shared across the Redis mesh.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit.*

#### 🧐 Reading the Signal:
- **Zero-Deficit Zone**: Because the broadcast work is now partially handled by Redis, the individual chat servers have more CPU headroom to handle WebSocket frames, significantly reducing the "Silent Failure" rate seen in previous labs.

---

### 🔬 Key Lessons
- **Shared State is Mandatory**: You cannot build a distributed chat without a shared bus.
- **Redis Overhead**: While Redis adds a small network hop tax, the **Scalability Gain** far outweighs the few milliseconds of latency it introduces.

---

### 🚀 Commands
```bash
# Start the lab (2 Replicas)
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-03-redis-pubsub/benchmark/run.py
```

---
[Next Lab: Lab 04 (Scalable Monolith) ➡️](../lab-04-scalable-monolith/README.md)
