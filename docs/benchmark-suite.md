# ChatLab Standard Benchmark Suite

ChatLab now uses one standard benchmark suite for fair architecture comparison across labs.

## Standard Comparable Run

The canonical fair-comparison workload is `comparison_standard`.

It is shared across labs and is intended to answer one question consistently:
"How does this architecture behave under the same steady interactive chat workload?"

The comparable run uses:
- 100 virtual users
- 1 minute duration
- 1000 ms message interval
- 256-byte payload target
- one fixed room identifier

## Standard Output Metrics

Every comparable run is summarized with the same three top-line metrics:
- latency: p50, p90, p95, and p99 end-to-end latency
- throughput: average and peak messages per second
- error rate: dropped or failed work relative to processed traffic

Supporting metrics such as DB latency, delivery ratio, duplicates, queue lag, or memory are still recorded when a lab exposes them.

## Why One Suite Matters

Without one shared comparison workload, cross-lab conclusions become fuzzy because changes in load shape can explain the result instead of architecture.

The standard suite keeps these fixed:
- client pacing
- message size
- concurrency target
- measurement semantics
- output artifact structure

## Running The Suite

Run the comparable suite across every benchmark-enabled lab:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard
```

By default this command excludes Lab 11 so the suite stays fast for routine local checks.
Include the capstone explicitly when needed:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard --include-blueprint
```

Run the comparable suite for one lab:

```bash
python3 scripts/chatlab.py bench lab-05-cloud-native-chat-infrastructure --scenario comparison_standard
```

Rebuild the aggregate report:

```bash
python3 scripts/chatlab.py report
```

Run local quality gates:

```bash
python3 scripts/chatlab.py validate
```

## Deep-Dive Scenarios

Labs can still define extra scenarios for local exploration such as saturation, spike recovery, security overhead, or global scaling. Those scenarios are useful for understanding the special behavior of each lab, but they are not the baseline for fair cross-lab comparison.
