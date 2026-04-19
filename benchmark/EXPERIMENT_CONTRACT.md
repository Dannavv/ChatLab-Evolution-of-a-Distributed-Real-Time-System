# Experiment Contract

This document defines what must stay constant across benchmark runs versus what is allowed to vary.

## Goals

- Make runs reproducible and auditable.
- Keep cross-lab comparisons fair.
- Produce machine-readable run metadata for downstream analysis.

## Constants

- Load generator script: `k6/base.js`.
- Metric scrape endpoint: `http://localhost:<port>/metrics`.
- Scraped metric names:
  - `chat_active_connections`
  - `chat_message_latency_ms_sum`
  - `chat_message_latency_ms_count`
  - `chat_memory_bytes`
  - `chat_messages_total`
  - `chat_dropped_messages_total`
- Output artifacts per run:
  - `metadata.json`
  - `timeseries.csv`
  - `k6_summary.json`

## Variables

- Lab under test.
- Workload manifest in `benchmark/workloads/*.yaml`.
- WebSocket ingress ports used for load generation.
- Metrics ports used for scrape aggregation.

## Required Workload Fields

- `name`
- `duration_seconds`
- `warmup_seconds`
- `scrape_interval_seconds`
- `message_interval_ms`
- `stages` list with:
  - `duration`
  - `target_vus`

## Run Identity

Each run ID is generated as:

`<lab>__<workload>__<UTC timestamp>`

Example:

`lab-04-scalable-monolith__robust_steady__20260210T153000Z`

## Artifact Layout

Raw artifacts are written under:

`benchmark/results/raw/<run_id>/`

Compatibility copy is also written to:

`results/<lab>_robust_report.csv`

## Metadata Requirements

`metadata.json` must include:

- `run_id`, `lab`, `workload`
- `started_at_utc`
- `metrics_ports`, `ws_ports`
- Full `workload_config`
- `compose_digest_sha256`
- Environment info (`python`, `platform`, `host`)
- Git revision info (`git rev-parse HEAD`)

## Scope Note

This is the Phase 0/1 contract. Future phases will add strict hardware normalization and richer consistency/failure annotations per scenario.
