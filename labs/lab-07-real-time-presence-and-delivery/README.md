[🏠 Home](../../README.md) | [⬅️ Previous (Lab 06)](../lab-06-chaos-and-resilience/README.md) | [Next Lab (Lab 08) ➡️](../lab-08-global-multi-region/README.md)

# Lab 07: Real-Time Presence and Delivery
## *User State Synchronization and the Presence Tax*

### 🔬 The Hypothesis
> "Implementing real-time presence tracking (online/offline status) will significantly increase the 'Background Traffic' per connection. This architecture will prove that as the number of users in a room grows (N), the number of presence update messages grows exponentially (N^2), requiring specialized 'Presence Servers' to avoid saturating the main chat bus."

### 🔴 The Problem: The N^2 Broadcast
In previous labs, we only sent "Messages." 
- **The Challenge**: Users expect to see who is currently typing or online. 
- **The Scalability Wall**: If 100 users are in a room and 1 user joins, 100 presence updates must be sent. If 1,000 users are in a room, a single join triggers 1,000 updates. This is the **N^2 Scaling Problem**.

---

### 🏗️ Architecture
![Lab 07 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Presence-Aware Mesh. Chat Server <-> Redis (Presence Key Store) <-> Client.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh showing the impact of presence synchronization.*

#### 🧐 Reading the Signal:
1.  **Increased Baseline Latency**: Notice that even with low message volume, the "Latency" is higher than Lab 01. This is the cost of constant heartbeat checks.
2.  **The Presence Tax**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Latency vs. User Count. The curve is steeper than Lab 03 because the server is doing significantly more work per connection to sync state.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing "Presence Saturation."*

#### 🧐 Reading the Signal:
- **Packet Storms**: The red area in Figure 4 represents "Dropped Heartbeats." When the server becomes too busy broadcasting presence updates, it misses its own health-checks, leading to "Flapping" (users appearing to go offline/online repeatedly).

---

### 🔬 Key Lessons
- **Presence is Expensive**: Don't broadcast every status change globally. Use **Throttling** or **Presence Regions**.
- **The Value of Ephemeral State**: Presence data belongs in Redis (Memory), never in the main PostgreSQL database.

---

### 🚀 Commands
```bash
# Start the presence-aware lab
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-07-real-time-presence-and-delivery/benchmark/run.py
```

---
[Next Lab: Lab 08 (Global Multi-Region) ➡️](../lab-08-global-multi-region/README.md)
