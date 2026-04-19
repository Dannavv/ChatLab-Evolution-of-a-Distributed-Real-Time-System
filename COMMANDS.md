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
```bash
cd labs/lab-08-global-multi-region && docker-compose up --build -d
```

### Lab 09: Message Security and Trust
```bash
cd labs/lab-09-message-security && docker-compose up --build -d
```

### Lab 10: Microservices Migration
```bash
cd labs/lab-10-microservices-migration && docker-compose up --build -d
```

---

## 📈 2. Benchmarking (Manifest-Driven)
The orchestrator automates: startup -> load generation -> metrics scrape -> teardown.

### Interactive Benchmark Launcher
```bash
python3 main.py
```

### Direct Run (Lab + Workload)
```bash
python3 benchmark/orchestrator.py lab-01-monolith-baseline robust_steady
```

### Available Workloads
```bash
ls benchmark/workloads
```

### Example Scenarios
```bash
python3 benchmark/orchestrator.py lab-04-scalable-monolith latency_probe
python3 benchmark/orchestrator.py lab-06-chaos-and-resilience spike_recovery
```

### Raw Artifact Location
```bash
ls benchmark/results/raw
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
*Aggregates latest raw run summaries (with fallback to legacy results) into a comparison table.*
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
