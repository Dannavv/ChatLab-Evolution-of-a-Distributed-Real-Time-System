[🏠 Home](../../README.md) | [⬅️ Previous (Lab 05)](../lab-05-cloud-native-chat-infrastructure/README.md) | [Next Lab (Lab 07) ➡️](../lab-07-real-time-presence-and-delivery/README.md)

# Lab 06: Chaos and Resilience
## *Circuit Breakers, Retries, and the Dead Letter Queue*

### 🔬 The Hypothesis
> "By implementing Circuit Breakers and Retry Policies with exponential backoff, we can prevent 'Cascading Failures.' The system will detect downstream service degradation and proactively shed load, ensuring the 'Core API' remains responsive even when the 'Persistence Worker' is failing."

### 🔴 The Problem: The Cascading Failure
In Lab 05, if the Worker was slow, the Redis Queue filled up. 
- **The Limit**: Eventually, the Queue hits the memory limit, and the API crashes. One bad component kills the whole system.
- **The Solution**: **Active Resilience**. The API monitors the health of the Worker. If the Worker fails too many times, the **Circuit Breaker trips (opens)**. The API stops sending to the failing service and instead routes to a **Dead Letter Queue (DLQ)**.

---

### 🏗️ Architecture
![Lab 06 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Resilient Mesh. API -> [Circuit Breaker] -> Worker -> [Dead Letter Queue].*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh under "Chaos" conditions (simulated worker failures).*

#### 🧐 Reading the Signal:
1.  **Latency Stabilization**: Notice the sharp "Spikes" in the Latency graph. These are the moments the Circuit Breaker is "Testing" the connection.
2.  **The Trip Proof**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Latency Profile. Note the "Plateau"—when the breaker is OPEN, latency is extremely low (fast-fail). When it is CLOSED, latency is higher as it attempts to process.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing "Self-Healing."*

#### 🧐 Reading the Signal:
- **DLQ Absorption**: The red area in Figure 4 is no longer "Lost Data." It represents messages that were successfully diverted to the **Dead Letter Queue**. Once the "Chaos" subsided and the worker recovered, these messages were automatically re-processed.

---

### 🔬 Key Lessons
- **Fast-Fail is Better than Slow-Hang**: Users prefer an "Error" to a "Loading Spinner" that never ends.
- **Observability is Resilience**: Without metrics showing the Breaker status, you are flying blind in a distributed storm.

---

### 🚀 Commands
```bash
# Start the lab with simulated chaos
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-06-chaos-and-resilience/benchmark/run.py
```

---
[Next Lab: Lab 07 (Real-Time Presence) ➡️](../lab-07-real-time-presence-and-delivery/README.md)
