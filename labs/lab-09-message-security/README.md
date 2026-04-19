[🏠 Home](../../README.md) | [⬅️ Previous (Lab 08)](../lab-08-global-multi-region/README.md) | [Next Lab (Lab 10) ➡️](../lab-10-microservices-migration/README.md)

# Lab 09: Message Security
## *E2EE, Key Rotation, and the Security Tax*

### 🔬 The Hypothesis
> "Implementing End-to-End Encryption (E2EE) and frequent 'Key Rotations' will significantly increase the CPU utilization per message. This architecture will prove that while the system remains durable and distributed, the cryptographic overhead will reduce the maximum concurrent users (VUs) by >30% compared to the unsecured baseline."

### 🔴 The Problem: The Transparent Mesh
In previous labs, messages were sent in plain text.
- **The Risk**: Anyone with access to Redis or the Database can read private conversations.
- **The Solution**: **AES-GCM Encryption**. Messages are encrypted on the server (or client) using regional master keys. These keys are rotated every 10 seconds to minimize the impact of a potential breach.

---

### 🏗️ Architecture
![Lab 09 Architecture](assets/benchmarks/architecture.png)
*Figure 1: The Secure Mesh. Message -> [AES-GCM Encrypt] -> Redis -> [AES-GCM Decrypt] -> Client.*

---

### 📊 Performance Analysis
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
*Figure 2: Performance mesh showing the "Security Tax" under load.*

#### 🧐 Reading the Signal:
1.  **CPU Spike**: Notice the "Memory" and "Processing" graphs are higher than Lab 03. Cryptography is a CPU-intensive operation.
2.  **The Rotation Penalty**:
   ![Latency Scaling](assets/benchmarks/modern_latency_scaling.png)
   *Figure 3: Latency Profile. You will see recurring "Jitter" or spikes every 10 seconds. This is the **Key Rotation Window**—the moment the system generates new secrets and propagates them across the mesh.*

---

### 📉 Reliability Audit
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
*Figure 4: Throughput Deficit showing "Cryptographic Saturation."*

#### 🧐 Reading the Signal:
- **Throughput Decay**: The red area in Figure 4 shows where the CPU can no longer keep up with the encryption/decryption demands. The "Security Tax" has effectively lowered our system's maximum capacity.

---

### 🔬 Key Lessons
- **Security is a Resource**: You cannot have E2EE for free. You must budget for the additional CPU cycles.
- **Rotation Frequency Trade-off**: Shorter rotation windows increase security but create more "System Jitter."

---

### 🚀 Commands
```bash
# Start the secure chat stack
docker-compose up --build -d

# Run local benchmark
python3 labs/lab-09-message-security/benchmark/run.py
```

---
[Next Lab: Lab 10 (Microservices Migration) ➡️](../lab-10-microservices-migration/README.md)
