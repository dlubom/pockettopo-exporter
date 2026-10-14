#!/usr/bin/env python3
"""Verify frozen prior and P14 evidence; never update expectations in a gate."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def verify(items, root):
    paths = [x["path"] for x in items]
    if not paths or len(set(paths)) != len(paths):
        raise ValueError("empty or duplicate evidence inventory")
    for item in items:
        path = root / item["path"]
        if not path.is_file() or path.is_symlink() or root.resolve() not in path.resolve().parents:
            raise ValueError("invalid evidence path: " + item["path"])
        if hashlib.sha256(path.read_bytes()).hexdigest() != item["sha256"]:
            raise ValueError("frozen evidence differs: " + item["path"])


if __name__ == "__main__":
    for name in ("preserved-evidence.json", "p14-evidence.json", "p15-evidence.json", "p16-evidence.json"):
        items = json.loads((ROOT / "validation" / name).read_text())
        verify(items, ROOT)
        print("Verified", name, len(items), "identities")
