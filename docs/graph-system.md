# Benchmark Graph System

## Overview

ChatLab now automatically generates and manages benchmark graphs at two levels:

1. **Per-run graphs** - Generated for each benchmark run in the lab-specific `benchmark/results/<run_id>/graphs/` directory
2. **Suite-level graphs** - Generated in `/assets/benchmarks/` directory for use in README files

## Graph Types

Each benchmark run generates three graphs:

1. **modern_latency_scaling.png** - Shows E2E latency trend over time with smoothing
   - X-axis: Time (seconds)
   - Y-axis: Latency (milliseconds, log scale)
   - Raw data + 15-point moving average

2. **modern_reliability_loss.png** - Shows throughput deficit relative to expected rate
   - X-axis: Time (seconds)  
   - Y-axis: Messages/sec lost
   - Filled area shows reliability degradation

3. **modern_quad_dashboard.png** - 2x2 dashboard showing:
   - Top-left: Latency trend
   - Top-right: Virtual user count (workload)
    - Bottom-left: Throughput (messages per second)
   - Bottom-right: Memory usage

## Prometheus Metric Default Behavior

Some labs do not export DB latency instrumentation (`chat_db_query_duration_ms_sum` and `chat_db_query_duration_ms_count`).

In that case, Prometheus scrapes used by the benchmark pipeline effectively produce default-zero values for DB latency in generated timeseries data. This is expected behavior and does not indicate benchmark failure.

To avoid misleading visuals across mixed lab architectures, SQL-overhead charting is intentionally not shown in the current graph set.

## Automatic Graph Generation

Graphs are automatically generated in three scenarios:

### 1. During individual benchmark runs
```bash
python3 scripts/chatlab.py bench lab-01-monolith-baseline --scenario comparison_standard
```
→ Generates per-run graphs in `labs/lab-01-monolith-baseline/benchmark/results/lab01__comparison_standard__<timestamp>/graphs/`

### 2. During suite runs
```bash
python3 scripts/chatlab.py suite --scenario comparison_standard
```
→ Generates per-run graphs for each lab
→ Generates suite-level graphs in `/assets/benchmarks/` from latest comparison_standard run

### 3. Via main.py interactive menu
```bash
python3 main.py
# Select: 0 (Run all labs)
```
→ Runs all benchmarks
→ Generates all graphs (per-run and suite-level)
→ Rebuilds comparison report

## Manual Graph Regeneration

To regenerate all graphs from existing benchmark data:

```bash
python3 scripts/chatlab.py regenerate-graphs
```

This:
- Regenerates all per-run graphs from existing timeseries.csv files
- Regenerates suite-level graphs in `/assets/benchmarks/` from latest runs
- Useful after recovering corrupted graph files

## README Integration

Each lab's README.md file references benchmark graphs using relative paths:

```markdown
![Modern Dashboard](assets/benchmarks/modern_quad_dashboard.png)
![Reliability Loss](assets/benchmarks/modern_reliability_loss.png)
```

These graphs are:
- Stored in `labs/<lab>/assets/benchmarks/` directory
- Updated automatically during benchmark runs
- Used to display latest performance data in GitHub/docs renderers

## Graph Data Flow

```
Benchmark Run (k6 + Prometheus scraping)
    ↓
benchmark/results/<run_id>/timeseries.csv
    ↓
_prepare_frame() — Parse & validate numeric columns
    ↓
Per-run graphs (3 files)
    → benchmark/results/<run_id>/graphs/
    
Suite-level graphs (3 files)
    → assets/benchmarks/
    
README.md references
    → Display latest graphs
```

## Data Validation

The plotting system includes robust data validation:

- **_ensure_numeric_columns()** - Converts CSV columns to numeric, handling parse errors
- **_prepare_frame()** - Validates timeseries.csv structure before plotting
- **Error handling** - Skips malformed runs with warnings, doesn't crash

## Troubleshooting

### Graphs not updating after benchmark
```bash
python3 scripts/chatlab.py regenerate-graphs
```

### Missing timeseries.csv
- This happens if benchmark was interrupted before metrics collection completed
- Re-run the benchmark: `python3 scripts/chatlab.py bench <lab-name> --scenario comparison_standard`

### PNG rendering issues in GitHub
- GitHub caches images for 24+ hours
- Force refresh by appending query string: `?t=<timestamp>`
- Or open raw PNG in new tab for immediate view

### Suite graphs not in assets/benchmarks/
```bash
python3 scripts/chatlab.py report  # Rebuilds report
python3 scripts/chatlab.py regenerate-graphs  # Regenerates all graphs
```

## Performance

Graph generation is fast (~0.5-1s per benchmark run):

- CSV parsing: ~100ms
- Per-run graphs (3 plots): ~300ms
- Suite-level graphs (3 plots): ~200ms

Total suite generation (all labs): ~5-10 seconds

## Integration with CI/CD

To integrate with automated workflows:

```bash
# After benchmark suite
python3 scripts/chatlab.py suite --scenario comparison_standard
# Graphs are automatically generated and comparison.md is updated

# Or manually ensure graphs are fresh
python3 scripts/chatlab.py regenerate-graphs
```
