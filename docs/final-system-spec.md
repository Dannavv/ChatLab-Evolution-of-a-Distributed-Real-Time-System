# ChatLab Final System Spec

This document defines the capstone target architecture for the ChatLab curriculum. It serves as the reference point that ties the labs into one coherent system-design story and now maps directly to the **Lab 11 Hardened Production Blueprint**.

## Goal

Build a real-time chat platform that:
- delivers interactive messages with low tail latency
- preserves durable history
- tolerates component and network failure better than a single node
- exposes enough telemetry to explain latency, backlog, and delivery outcomes
- makes security, cost, and consistency trade-offs explicit

## Implemented Service Topology

```text
Client
  -> API gateway / websocket edge
     -> auth and rate-limit layer
     -> regional chat ingress
        -> room routing / session registry
        -> durable message write path
        -> broker or queue handoff
        -> fan-out workers
        -> delivery and presence services
        -> history service
        -> object/archive storage
  -> observability stack
     -> Prometheus metrics
     -> traces
     -> structured logs
```

## Core Responsibilities

| Component | Responsibility | Main trade-off |
| --- | --- | --- |
| Gateway | Terminate client traffic, authenticate users, enforce rate limits | Adds an extra hop but centralizes control |
| Chat ingress | Accept messages and attach trace metadata | Must stay thin to protect latency |
| Broker / queue | Decouple bursty ingest from slower downstream work | Backlog can hide trouble until it grows |
| Fan-out workers | Deliver messages to active websocket sessions | Fast delivery depends on routing locality |
| Presence service | Track ephemeral online state and typing | Freshness matters more than durability |
| History service | Store durable message history and replay | Write durability raises latency budget |
| Multi-region bridge | Replicate events across regions asynchronously | Cross-region visibility becomes eventual |
| Key management | Rotate secrets and support encrypted paths | Security adds CPU and coordination overhead |
| Observability stack | Explain tail latency, retries, backlog, and failures | Telemetry itself has runtime and ops cost |

## Target SLOs

These are suggested capstone targets for local or single-environment validation:

| SLI | Target |
| --- | --- |
| Single-region p95 end-to-end latency | `< 50 ms` |
| Single-region p99 end-to-end latency | `< 100 ms` |
| Accepted-message delivery ratio | `>= 99.9%` |
| Benchmark error rate | `< 0.1%` |
| Regional failover recovery visibility | `< 60 s` for degraded but functioning service |
| Queue backlog drain after burst | returns to baseline within the workload recovery window |

For multi-region traffic, the user-visible goal should be explicit: local-region delivery should remain interactive while remote-region convergence is allowed to be eventual.

## Consistency Model

The capstone system uses a mixed model:
- message acceptance is durable within the local write domain
- active websocket fan-out is near-real-time but may be retried
- presence is ephemeral and eventually convergent
- cross-region replication is asynchronous
- history replay is authoritative for durable state

This means the system should document where read-after-write is guaranteed and where users may observe short-lived divergence.

## Failure Model

The final system should be able to describe behavior under at least these scenarios:
- websocket node crash during active fan-out
- database slowdown or transient write failure
- broker lag or temporary disconnect
- worker crash with messages still queued
- regional bridge lag or isolation
- key rotation mismatch or decrypt failure
- rate-limit enforcement during abuse spikes

For each case, the desired outcome should be stated in terms of:
- message loss boundary
- retry behavior
- user-visible degradation
- recovery signal in metrics and traces

## Observability Requirements

Minimum capstone telemetry should include:
- Prometheus counters and histograms for ingress, persistence, broker, retry, and fan-out paths
- propagated `trace_id` and `message_id` across every hop
- distributed traces for ingress, queue handoff, DB write, broadcast, and replay
- queue depth, retry count, dead-letter count, and connection count
- latency distribution tracking at p50, p95, and p99

The key question observability must answer is not just "is it up?" but "why did tail latency or delivery quality change?"

## Security Expectations

The final system should treat security as a system property, not only a crypto feature:
- authenticated websocket or HTTP session establishment
- authorization checks for room membership and admin actions
- rate limiting at ingress
- secret management and rotation discipline
- encrypted message or transport path where the lab requires it
- replay defense using stable message identity
- basic OWASP-style hardening for web-facing components

## Testing Matrix

The repo should eventually support a layered test story:

| Test type | What it validates |
| --- | --- |
| Unit tests | local parsing, routing, queue, auth, and crypto logic |
| Integration tests | service-to-service behavior with broker, DB, and storage |
| Load tests | latency, throughput, backlog, and error-rate behavior |
| Chaos tests | **(Implemented)** Automated retries, circuit breakers, and Recovery Time Objective (RTO) |
| Security tests | auth bypass, replay handling, rate limiting, and key misuse cases |

## Cost Model

A complete capstone evaluation should compare:
- single-node CPU and memory cost
- database write amplification and storage growth
- broker and queue infrastructure cost
- websocket session density per node
- cross-region egress overhead
- encryption and signing CPU tax
- operational complexity introduced by service decomposition

The right outcome is not "the most distributed system." It is the system whose added cost buys a clear reliability, latency, or operational benefit.

## Advanced Extensions

Natural follow-on labs or capstone extensions include:
- event sourcing for durable append-only history
- CQRS for read-optimized timelines and delivery views
- partition-aware sharding by room, tenant, or geography
- replica lag instrumentation and consistency experiments
- formal backpressure budgets and admission control
- tenant isolation and RBAC

## What This Spec Resolves

This document gives the curriculum a concrete finish line:
- a final architecture, not just a sequence of demos
- measurable reliability and latency goals
- explicit security and testing expectations
- clear acknowledgement of trade-offs and remaining gaps
