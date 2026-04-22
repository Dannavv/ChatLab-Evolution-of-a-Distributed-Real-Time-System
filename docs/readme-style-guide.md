# ChatLab README Style Guide

This guide keeps tone, clarity, and polish consistent across labs.

## Tone

- Write in clear technical prose: direct, concrete, and evidence-oriented.
- Prefer architecture and operational language over marketing language.
- Avoid vague claims; tie statements to measured behavior.

## Writing Rules

- Start with user-visible stakes in `## Hook`.
- Use measurable verbs in `## Learning Outcomes` (measure, compare, explain, identify).
- Use production framing in `## Why This Matters in Production`.
- Keep section openings concise (1 to 3 sentences before bullets).

## Section Intent

- `Hook`: why this lab is worth running now.
- `Learning Outcomes`: what skills the reader will gain.
- `Why This Matters in Production`: decision relevance under real constraints.
- `Expected Result`: what signals should appear when run succeeds.

## Formatting

- Use one blank line between headings and content.
- Use flat bullet lists; avoid nested bullets unless absolutely needed.
- Keep code blocks focused and runnable.
- Keep line wrapping human-readable for markdown diffs.

## Evidence-First Language

Prefer:
- "Measure p95 and throughput under the same scenario"
- "Observe queue lag before user-visible errors"

Avoid:
- "This architecture is best"
- "This should scale infinitely"

## Consistency Gate

Use the README validator before publishing edits:

```bash
python3 scripts/chatlab.py validate --kind readmes
```
