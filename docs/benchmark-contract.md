# ChatLab Benchmark Contract

This document defines the repo-wide benchmark contract that all labs now share.

## Why This Exists

The curriculum evolves one chat system through ten architectural stages. To keep the learning arc cumulative, every lab now declares the same benchmark shape and the same interpretation rules for workload, metrics, consistency, routing, failure handling, observability, traceability, and cost.

## Shared Comparison Scenario

Every lab exposes `comparison_standard` with the same baseline assumptions:

- 100 virtual users
- 1 minute duration
- 1000 ms message interval
- 256-byte payload target
- fixed room identifier for repeatability

This is the scenario used by [results/comparison.md](../results/comparison.md) and by the standardized repo-level command:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard
```

This single comparable run is the standard benchmark suite for fair architecture comparison. The summary lens is always:
- latency
- throughput
- error rate

## Global Metric Definitions

- `latency`: client-observed end-to-end latency, computed as `client_receive_ts - client_send_ts`
- `throughput`: processed messages per second derived from Prometheus counters
- `error_rate`: dropped messages plus DB errors divided by processed messages
- `db_latency`: database latency when a lab has a persistence path
- `delivery_ratio`: received messages divided by sent messages
- `duplicate_ratio`: duplicate messages divided by sent messages

## Traceability Contract

All labs now declare the same traceability surface:

- `trace_id`: the logical request trace
- `message_id`: the message identity used for duplicate detection and replay defense
- `client_send_ts`: load-driver send timestamp
- `server_receive_ts`: first server ingress timestamp when the lab emits it
- `server_broadcast_ts`: fan-out timestamp when the lab emits it

The goal is that each later lab makes message identity explicit even as components multiply.

## Consistency Progression

- Lab 01: immediate single-node visibility
- Lab 02: strong single-writer durability
- Lab 03: eventual multi-node convergence
- Lab 04: immediate single-node visibility with explicit queueing
- Lab 05: asynchronous durable processing with eventual visibility
- Lab 06: eventual durability with resilience controls
- Lab 07: mixed model with eventual presence convergence
- Lab 08: eventual cross-region convergence
- Lab 09: eventual secure visibility
- Lab 10: service-coordinated durability with eventual multi-service visibility
- Lab 11: service-coordinated durability with operational guardrails

## Routing Progression

- direct single-node routing
- synchronous write-through
- pub/sub fan-out
- ingress queue plus worker pool
- API plus worker pipeline
- resilient worker pipeline
- stateful websocket routing
- latency-based regional affinity with asynchronous bridging
- secure ingress validation
- gateway-mediated service routing
- gateway-mediated service routing with standardized control-plane tooling

## Failure Progression

Each workload manifest includes an explicit `failure_model` with a focus and concrete mechanisms, so failure handling is part of the benchmark story instead of only an implementation detail.

## Observability Baseline

Every lab is expected to expose:

- Prometheus metrics
- Docker Compose logs
- Grafana dashboard support through the shared provisioning bundle
- benchmark artifacts under `benchmark/results/<run_id>/`
- `metadata.json`
- `timeseries.csv`
- `benchmark_summary.json`
- generated graphs

## Cost Awareness

Each lab now declares a `cost_model` so the comparison report can tie architectural choices to their dominant resource axis:

- single-node CPU and memory
- database IOPS
- broker/network cost
- worker and queue cost
- retry amplification
- websocket session density
- cross-region egress
- cryptographic CPU overhead
- service-to-service coordination overhead

## Comparison Synthesis

`shared/benchmark/report.py` builds:

- [results/comparison.md](../results/comparison.md)
- `results/comparison.json`

These artifacts summarize the latest `comparison_standard` run from each lab and turn the repo into one cumulative benchmark narrative instead of ten disconnected experiments.

The report now also includes a final architecture comparison table covering:
- complexity
- dominant cost axis
- scalability posture
- failure handling posture
- real-world architectural mapping
