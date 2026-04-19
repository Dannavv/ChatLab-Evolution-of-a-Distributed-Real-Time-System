[🏠 Home](../../README.md) | [⬅️ Previous (Lab 03)](../lab-03-redis-pubsub/README.md) | [Next Lab (Lab 05) ➡️](../lab-05-cloud-native-chat-infrastructure/README.md)

# Lab 04: The Scalable Monolith
## *Vertical Scaling and Internal Worker Pools*

**Purpose:** protect the hot path of a single node by introducing internal queues and worker pools.  
**Hypothesis:** queueing work behind workers will stabilize throughput during bursts, but it will convert synchronous blocking into visible queueing latency.

### 🎯 Objective
This lab keeps the system on one node but changes how that node absorbs load. The goal is to prove that a carefully controlled queue and worker pool can prevent the WebSocket handler from doing all expensive work inline.

### 🔁 What Changed From Previous Lab
- Lab 03 distributed fan-out across multiple nodes, but each node still handled work immediately.
- Lab 04 re-centers on single-node protection by introducing internal producer-consumer behavior.
- The request path now includes a queue handoff before heavy processing completes.
- This lab focuses on controlled degradation instead of purely horizontal fan-out.

### 🔬 The Hypothesis
> "By introducing an internal worker pool and a message queue, we can prevent 'Synchronous Blocking' and absorb high-concurrency spikes. This architecture will maintain stable throughput even when the processing logic is heavy, at the cost of increased 'Queueing Latency'."

### 🔴 The Problem: The Blocking Handler
In previous labs, the server processed every message immediately. 
- **The Limit**: If a message takes 50ms to process (simulated work), the server can only handle 20 messages per second per thread. Under load, this causes a "Chain Reaction" of timeouts.
- **The Solution**: **Producer-Consumer Pattern**. The WebSocket handler "produces" to a queue, and a pool of workers "consume" at their own pace.

---

### 🏗️ Architecture
![Lab 04 Architecture](assets/benchmarks/architecture.png)
*Figure 1: Internal scaling via Worker Pools. This is the foundation of high-performance Go services.*

### 🏛️ System Architecture (Structured View)
```text
Client
  -> WebSocket handler
     -> enqueue work
     -> worker pool processes queued messages
     -> completed messages are broadcast to clients
```

### 🔄 Request Flow
1. The client sends a message to the chat server.
2. The WebSocket handler accepts the message and pushes it into an internal queue.
3. A worker pulls the message from the queue and performs the heavier processing path.
4. The node broadcasts the completed result.
5. If the queue is saturated, the system degrades in a controlled way rather than blocking the entire handler path.

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Unified view of the worker-pool performance under stress.*

#### 🧐 Reading the Signal:
1.  **Stable Throughput**: Notice how the "Throughput" graph remains a flat line even when users spike. The worker pool is "Gating" the traffic to protect the system.
2.  **The Queueing Penalty**: 
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Latency Profile. Note the "Staircase" effect—as the queue fills up, latency increases because messages are waiting longer for a free worker.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit.*

#### 🧐 Reading the Signal:
- **Graceful Degradation**: Unlike Lab 01 (which crashed), Lab 04 shows a **Predictable Deficit**. The red area represents the "Queue Overflow"—we are intentionally dropping messages once the queue is full to prevent the server from running out of memory.

### 🧪 Benchmark Notes
- Benchmark README: [benchmark/README.md](./benchmark/README.md)
- Main benchmark scenario: `scaling_monolith`
- Direct run command:
```bash
python3 labs/lab-04-scalable-monolith/benchmark/run.py --scenario scaling_monolith
```

### 🧾 Interpretation
Performance changes here because the system now protects itself with admission control. The flatter throughput line is a feature, not an accident: the queue is smoothing the burst, while latency grows because messages wait longer before a worker becomes available.

### 🚧 Limitations
- The system is still bound to one node's memory and CPU.
- Queueing protects the process, but it does not eliminate backlog.
- Intentional drops are safer than collapse, but they are still user-visible loss.

---

### 🔬 Key Lessons
- **Queues Save Lives**: A system without a queue is a system waiting to fail.
- **The Limit of Vertical Scaling**: While worker pools help, a single node still has a physical memory limit. True scale requires Lab 05 (Cloud-Native Infrastructure).

### ✅ What This Enables For Next Lab
Lab 04 teaches us how to protect a node, but not how to split responsibilities cleanly. Lab 05 takes the next step by separating ingest from background processing and long-term storage.

---

### 🚀 Commands
```bash
# Start the lab
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-04-scalable-monolith/benchmark/run.py
```

---
[Next Lab: Lab 05 (Cloud-Native Chat Infrastructure) ➡️](../lab-05-cloud-native-chat-infrastructure/README.md)
