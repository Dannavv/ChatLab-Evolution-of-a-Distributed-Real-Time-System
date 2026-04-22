# ChatLab README Contract

This contract defines the required structure for documentation consistency across the project.

## Scope

- Root README: `README.md`
- Lab READMEs: `labs/lab-*/README.md`

## Root README Required Sections

The root README must include these sections in this order:

1. `## Hook`
2. `## Learning Outcomes`
3. `## Why This Matters in Production`
4. `## 🛠️ Quick Start (The "System Owner" Way)`
5. `## 📊 System Evolution Summary`
6. `## 🎓 Curriculum Path`

## Lab README Required Sections

Every lab README must include these sections in this order:

1. `## Hook`
2. `## Learning Outcomes`
3. `## Why This Matters in Production`
4. `## Overview`
5. `## Architecture`
6. `## How to Run`
7. `## What Changed From Previous Lab`
8. `## Results`
9. `## Limitations`
10. `## Known Issues`
11. `## When This Architecture Fails`
12. `## Folder Structure`

## Lab README Additional Requirements

- Must contain `### Expected Result` under run guidance.
- Must contain at least one architecture visual:
  - Mermaid block (` ```mermaid `), or
  - image reference (`![...](...)`).
- `## Learning Outcomes` must contain at least 3 bullet points.

## Validation Command

Run all gates including README conformance:

```bash
python3 scripts/chatlab.py validate
```

Run README-only checks:

```bash
python3 scripts/chatlab.py validate --kind readmes
```
