# ChatLab Local Testing And Validation Guide

This guide defines the local quality-gate flow for ChatLab.

## Goals

- verify workload contract integrity before benchmark runs
- verify comparison report consistency after benchmark runs
- evaluate benchmark outputs against capstone SLO thresholds

## Recommended Order

1. Validate workload contracts
2. Run benchmark suite or a targeted benchmark
3. Rebuild the comparison report
4. Validate report consistency
5. Validate SLO outcomes

## Commands

Run all validation gates:

```bash
python3 scripts/chatlab.py validate
```

Run a specific gate:

```bash
python3 scripts/chatlab.py validate --kind workloads
python3 scripts/chatlab.py validate --kind results
python3 scripts/chatlab.py validate --kind slos
```

Routine comparable suite (without blueprint):

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard
```

Include the capstone blueprint:

```bash
python3 scripts/chatlab.py suite --scenario comparison_standard --include-blueprint
```

Run only the capstone comparable benchmark:

```bash
python3 scripts/chatlab.py bench lab-11-production-grade-blueprint --scenario comparison_standard
```

Rebuild comparison artifacts:

```bash
python3 scripts/chatlab.py report
```

## Output Interpretation

Workload validation:
- PASS means all workload manifests satisfy required keys and include comparison_standard scenario details.
- FAIL means one or more manifests violate the benchmark contract.

Results validation:
- PASS means comparison markdown and json are coherent and reliability status text matches available metrics.
- WARNING can appear when sent and received counters are unavailable.
- FAIL means contradictory or malformed report data.

SLO validation:
- PASS means all strict thresholds pass.
- WARN means soft issues such as unavailable delivery counters or missing comparable run.
- FAIL means hard reliability or latency thresholds are exceeded.

## Troubleshooting

If workload validation fails:
- open the reported workload file and add missing contract keys.

If results validation fails:
- rebuild report and rerun validation.
- check recent benchmark run artifacts for missing benchmark_summary.json or k6_summary.json.

If SLO validation fails:
- inspect latest benchmark summary in the corresponding lab results directory.
- compare p95, p99, and error rate values against capstone targets.

## Artifact Policy

Generated benchmark artifacts are local run outputs and should not be committed by default.
