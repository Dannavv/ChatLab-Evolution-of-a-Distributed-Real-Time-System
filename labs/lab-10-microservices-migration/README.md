[🏠 Home](../../README.md) | [⬅️ Previous (Lab 09)](../lab-09-message-security/README.md)

# Lab 10: Microservices Migration
## *The Mesh, The Gateway, and Service Isolation*

### 🔬 The Hypothesis
> "By migrating to a full Microservices Architecture, we can isolate failures and scale 'Reads' (History) independently from 'Writes' (Messaging). This architecture will prove that while the 'Network Tax' increases baseline latency, the overall 'System Reliability' is superior because a failure in the History Service will not affect the Real-Time Messaging Gateway."

### 🔴 The Problem: The Scaling Monolith
In previous labs, our "Server" still did everything.
- **The Limit**: If 10,000 users joined a room and started fetching "History," the server's CPU would spike, causing "Real-Time" messages to lag.
- **The Solution**: **Service Isolation**.
  - **API Gateway**: Handles only WebSocket connections and Auth.
  - **Message Service**: Handles only incoming "Sends."
  - **History Service**: Handles only database "Reads."

---

### 🏗️ Architecture
![Lab 10 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Microservices Mesh. Client -> Gateway -> [Service Discovery] -> Message/History Service.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh showing the "Service Isolation" benefits.*

#### 🧐 Reading the Signal:
1.  **The Network Tax**: Notice that the "Baseline Latency" is higher than any other lab. This is because every message now traverses the network **3-4 times** (Client -> Gateway -> Message Service -> Redis -> Client).
2.  **Independent Scaling Proof**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Latency Profile. Note how the "Real-Time" latency remains stable even when we flood the "History Service" with read requests. This is **Failure Isolation** in action.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing "Microservice Resilience."*

#### 🧐 Reading the Signal:
- **Zero-Coupling**: In Figure 4, you can see that even when one service is saturated, the others maintain their throughput. We have successfully broken the "Fate Sharing" of the monolith.

---

### 🔬 Key Lessons
- **Microservices are an Organizational Tool**: They allow teams to work independently, but they come with a **Performance Cost**.
- **The Gateway is the King**: In a mesh, the Gateway is your most critical scaling point. If the Gateway lags, the whole world lags.

---

### 🚀 Commands
```bash
# Start the full microservices mesh
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-10-microservices-migration/benchmark/run.py
```

---
[🏠 Return to Project Home](../../README.md)
