[🏠 Home](../../README.md) | [⬅️ Previous (Lab 07)](../lab-07-real-time-presence-and-delivery/README.md) | [Next Lab (Lab 09) ➡️](../lab-09-message-security/README.md)

# Lab 08: Global Multi-Region
## *Geographic Latency and the Regional Bridge*

### 🔬 The Hypothesis
> "By using Regional Redis clusters and a specialized 'Global Bridge,' we can maintain sub-10ms latency for users in the same region while ensuring global delivery. This architecture will prove that cross-region latency is bounded by the speed of light, and our goal is to minimize the 'Synchronous Wait' for distant regions."

### 🔴 The Problem: The Global Latency Wall
In previous labs, all users were in one "Data Center."
- **The Reality**: A user in Europe (EU) should not wait for a server in the USA (US) to acknowledge their message.
- **The Solution**: **Regional Isolation**. Users connect to their local region. A "Bridge Service" asynchronously syncs messages between the US and EU clusters.

---

### 🏗️ Architecture
![Lab 08 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Global Mesh. US Cluster <-> Global Bridge <-> EU Cluster.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh showing Regional vs. Global latency distribution.*

#### 🧐 Reading the Signal:
1.  **The Regional Speed Trap**: Notice the "Local" latency is extremely low. This proves our regional clusters are working independently.
2.  **The Bridge Bottleneck**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Global Latency Profile. You will see a distinct "Step" in latency (~150ms+). This is the simulated physical distance between the US and EU.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing "Bridge Saturation."*

#### 🧐 Reading the Signal:
- **Asynchronous Lag**: The deficit in Figure 4 represents the **Sync Lag**. If the bridge cannot keep up with the global message volume, users in the EU will see US messages with a delay. The red area shows where the bridge buffer is beginning to overflow.

---

### 🔬 Key Lessons
- **Speed of Light is the Hardest Constraint**: You can scale CPU, but you can't scale the speed of a fiber optic cable across the Atlantic.
- **Eventual Consistency is Mandatory**: At a global scale, synchronous writes are impossible. You must embrace asynchronous delivery.

---

### 🚀 Commands
```bash
# Start the global multi-region stack
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-08-global-multi-region/benchmark/run.py
```

---
[Next Lab: Lab 09 (Message Security) ➡️](../lab-09-message-security/README.md)
