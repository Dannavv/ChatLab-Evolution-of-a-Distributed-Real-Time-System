# 🛠️ ChatLab Control Commands

This reference guide contains the essential commands for running, benchmarking, and analyzing the ChatLab architectures.

---

## 🏗️ 1. Running the Labs
Each lab can be run independently using Docker Compose.

### Lab 01: Monolith Baseline
```bash
cd labs/lab-01-monolith-baseline && docker-compose up --build -d
```

### Lab 02: Persistence Layer (PostgreSQL)
```bash
cd labs/lab-02-persistence-layer && docker-compose up --build -d
```

### Lab 03: Distributed Pub/Sub (Redis)
```bash
cd labs/lab-03-redis-pubsub && docker-compose up --build -d
```

### Lab 04: Scalable Monolith
```bash
cd labs/lab-04-scalable-monolith && docker-compose up --build -d
```

### Lab 05: Cloud-Native Chat Infrastructure
```bash
cd labs/lab-05-cloud-native-chat-infrastructure && docker-compose up --build -d
```

### Lab 06: Chaos and Resilience
```bash
cd labs/lab-06-chaos-and-resilience && docker-compose up --build -d
```

### Lab 07: Real-Time Presence and Delivery
```bash
cd labs/lab-07-real-time-presence-and-delivery && docker-compose up --build -d
```

### Lab 08: Global Distribution
*Coming soon.*

### Lab 09: Message Security and Trust
```bash
cd labs/lab-09-message-security && docker-compose up --build -d
```

---

## 📈 2. Benchmarking (The Orchestrator)
The `orchestrator.py` script automates the full testing lifecycle (Start -> Load Test -> Scrape -> Shutdown).

### Standard Benchmark (10 VUs)
```bash
python3 benchmark/orchestrator.py --labs lab-01-monolith-baseline
```

### Scaling Test (10, 100, 500, 1000 VUs)
```bash
python3 benchmark/orchestrator.py --scaling --labs lab-01-monolith-baseline
```

### ☢️ Robust Mode (Smooth Ramp-up to 2500 VUs + Live Telemetry)
*Recommended for generating the "Manuscript" performance graphs.*
```bash
python3 benchmark/orchestrator.py --Robust-mode --labs lab-02-persistence-layer
```

### Lab 04 Benchmark Target
```bash
python3 benchmark/orchestrator.py --labs lab-04-scalable-monolith
```

### Lab 05 Benchmark Target
```bash
python3 benchmark/orchestrator.py --labs lab-05-cloud-native-chat-infrastructure
```

### Lab 06 Benchmark Target
```bash
python3 benchmark/orchestrator.py --labs lab-06-chaos-and-resilience
```

### Lab 07 Benchmark Target
```bash
python3 benchmark/orchestrator.py --labs lab-07-real-time-presence-and-delivery
```

### Lab 09 Benchmark Target
```bash
python3 benchmark/orchestrator.py --labs lab-09-message-security
```

---

## 📊 3. Analysis & Visualization
Use these commands to process the raw telemetry data into professional reports.

### Generate Master Performance Report (Landscape)
*Processes the latest Robust Mode CSV into a 3-panel landscape PNG.*
```bash
python3 benchmark/visualize.py
```

### Generate Comparison Report (Markdown)
*Aggregates all k6 JSON results into a single comparison table.*
```bash
python3 benchmark/generate-report.py
```

---

## 🧹 4. Maintenance & Cleanup

### Shutdown All Containers
```bash
docker stop $(docker ps -aq) && docker rm $(docker ps -aq)
```

### Clean Benchmark Assets
```bash
rm results/*.png results/*.csv results/*.json
```

### Hard Rebuild (Fix Telemetry/State Issues)
```bash
docker-compose down && docker-compose up --build -d
```
