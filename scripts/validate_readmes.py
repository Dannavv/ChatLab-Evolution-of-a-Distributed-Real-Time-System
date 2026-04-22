#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
LABS_DIR = ROOT_DIR / "labs"
ROOT_README = ROOT_DIR / "README.md"

ROOT_REQUIRED = [
    "## Hook",
    "## Learning Outcomes",
    "## Why This Matters in Production",
    "## 🛠️ Quick Start (The \"System Owner\" Way)",
    "## 📊 System Evolution Summary",
    "## 🎓 Curriculum Path",
]

LAB_REQUIRED = [
    "## Hook",
    "## Learning Outcomes",
    "## Why This Matters in Production",
    "## Overview",
    "## Architecture",
    "## How to Run",
    "## What Changed From Previous Lab",
    "## Results",
    "## Limitations",
    "## Known Issues",
    "## When This Architecture Fails",
    "## Folder Structure",
]


def _read(path):
    return path.read_text(encoding="utf-8")


def _find_heading_positions(text, headings):
    positions = {}
    for heading in headings:
        idx = text.find(heading)
        if idx != -1:
            positions[heading] = idx
    return positions


def _is_ordered(positions, headings):
    seen = [positions[h] for h in headings if h in positions]
    return seen == sorted(seen)


def _count_learning_bullets(text):
    marker = "## Learning Outcomes"
    start = text.find(marker)
    if start == -1:
        return 0
    next_heading = text.find("\n## ", start + len(marker))
    block = text[start: next_heading if next_heading != -1 else len(text)]
    return len(re.findall(r"(?m)^- ", block))


def _validate_root():
    result = {
        "file": str(ROOT_README.relative_to(ROOT_DIR)),
        "status": "PASS",
        "issues": [],
        "warnings": [],
    }

    if not ROOT_README.exists():
        result["status"] = "FAIL"
        result["issues"].append("README.md is missing")
        return result

    text = _read(ROOT_README)
    positions = _find_heading_positions(text, ROOT_REQUIRED)

    missing = [h for h in ROOT_REQUIRED if h not in positions]
    if missing:
        result["status"] = "FAIL"
        result["issues"].append("missing sections: " + ", ".join(missing))

    if not _is_ordered(positions, ROOT_REQUIRED):
        result["status"] = "FAIL"
        result["issues"].append("required sections are out of order")

    return result


def _validate_lab(path):
    rel = str(path.relative_to(ROOT_DIR))
    result = {
        "file": rel,
        "status": "PASS",
        "issues": [],
        "warnings": [],
    }

    text = _read(path)
    positions = _find_heading_positions(text, LAB_REQUIRED)

    missing = [h for h in LAB_REQUIRED if h not in positions]
    if missing:
        result["status"] = "FAIL"
        result["issues"].append("missing sections: " + ", ".join(missing))

    if not _is_ordered(positions, LAB_REQUIRED):
        result["status"] = "FAIL"
        result["issues"].append("required sections are out of order")

    if "### Expected Result" not in text:
        result["status"] = "FAIL"
        result["issues"].append("missing section: ### Expected Result")

    if "```mermaid" not in text and "![" not in text:
        result["status"] = "FAIL"
        result["issues"].append("missing architecture visual (mermaid or image)")

    bullet_count = _count_learning_bullets(text)
    if bullet_count < 3:
        result["status"] = "FAIL"
        result["issues"].append("Learning Outcomes must contain at least 3 bullets")

    if re.search(r"(?im)^i\s+think\b", text):
        result["warnings"].append("contains first-person speculative phrasing")

    return result


def main():
    reports = [_validate_root()]

    lab_readmes = sorted(LABS_DIR.glob("lab-*/README.md"))
    for path in lab_readmes:
        reports.append(_validate_lab(path))

    aggregate = {
        "checked": len(reports),
        "pass": sum(1 for r in reports if r["status"] == "PASS"),
        "fail": sum(1 for r in reports if r["status"] == "FAIL"),
        "warn": sum(1 for r in reports if r["warnings"]),
    }

    print(json.dumps({"aggregate": aggregate, "reports": reports}, indent=2))

    return 1 if aggregate["fail"] > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
