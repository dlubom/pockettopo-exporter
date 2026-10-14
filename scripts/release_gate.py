#!/usr/bin/env python3
"""Verify complete hosted CI and an optional annotated tag for one release SHA."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

REPO = Path(__file__).resolve().parent.parent
TARGETS = ("linux-amd64", "linux-arm64", "windows-amd64", "windows-arm64", "darwin-amd64", "darwin-arm64")


def validate_published(release, latest, tag):
    if (release.get("draft") is not False or release.get("prerelease") is not False
            or release.get("tag_name") != tag or not release.get("published_at")
            or type(release.get("id")) is not int or latest.get("id") != release["id"]
            or latest.get("tag_name") != tag):
        raise ValueError("release readback is not the published latest stable version")


def validate_assets(expected, downloaded):
    def inventory(directory):
        items = list(directory.iterdir())
        if not items or any(item.is_symlink() or not item.is_file() for item in items):
            raise ValueError("release assets must be nonempty regular files")
        return {item.name: hashlib.sha256(item.read_bytes()).hexdigest() for item in items}
    if inventory(expected) != inventory(downloaded):
        raise ValueError("downloaded release asset membership or bytes differ")


def validate_run(run, jobs, revision, expected):
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("invalid source revision")
    for field, value in {"name": "CI", "head_sha": revision, "head_branch": "main",
                         "event": "push", "status": "completed", "conclusion": "success"}.items():
        if run.get(field) != value:
            raise ValueError("release CI differs: " + field)
    names = [job.get("name") for job in jobs]
    if len(names) != len(set(names)) or not expected.issubset(set(names)):
        raise ValueError("release CI has duplicate or missing required jobs")
    if any(job.get("status") != "completed" or job.get("conclusion") != "success" for job in jobs):
        raise ValueError("release CI contains an unsuccessful or incomplete job")


def validate_tag(ref, tag, name, revision):
    if (ref.get("ref") != "refs/tags/" + name or ref.get("object", {}).get("type") != "tag"
            or tag.get("tag") != name or tag.get("object", {}).get("type") != "commit"
            or tag.get("object", {}).get("sha") != revision):
        raise ValueError("release tag is not annotated at the verified commit")


def run(args):
    return subprocess.run(args, cwd=REPO, check=True, capture_output=True, text=True, timeout=120).stdout


def api(repo, path, paginate=False):
    args = ["gh", "api", "repos/" + repo + "/" + path]
    if paginate:
        args.extend(["--paginate", "--slurp"])
    return json.loads(run(args))


def expected_jobs():
    groups = json.loads(run(["bash", "scripts/mutation.sh", "--matrix"]))["group"]
    return {"CI complete", "corpus", "walls-native", "survex-native", "mutation-plan",
            "check (ubuntu-24.04)", "check (windows-2025)", "check (macos-15)",
            *("mutation (" + group + ")" for group in groups),
            *("release-package (" + target + ")" for target in TARGETS)}


def verify(repo, run_id, revision, tag=None):
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo) or not re.fullmatch(r"[0-9]+", run_id):
        raise ValueError("invalid repository or run ID")
    result = api(repo, "actions/runs/" + run_id)
    pages = api(repo, "actions/runs/" + run_id + "/jobs?filter=latest&per_page=100", paginate=True)
    jobs = [job for page in pages for job in page["jobs"]]
    if sum(len(page["jobs"]) for page in pages) != pages[0]["total_count"]:
        raise ValueError("incomplete CI job response")
    validate_run(result, jobs, revision, expected_jobs())
    if tag:
        if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+", tag):
            raise ValueError("invalid stable release tag")
        ref = api(repo, "git/ref/tags/" + tag)
        tag_object = api(repo, "git/tags/" + ref["object"]["sha"])
        validate_tag(ref, tag_object, tag, revision)
    return {"schema_version": 1, "source_revision": revision, "run_id": int(run_id),
            "run_url": result["html_url"], "tag": tag, "all_jobs_success": True,
            "jobs": [{"name": job["name"], "conclusion": job["conclusion"]} for job in jobs]}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--revision", required=True)
    parser.add_argument("--tag")
    parser.add_argument("--published", action="store_true")
    parser.add_argument("--expected-assets", type=Path)
    parser.add_argument("--downloaded-assets", type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    evidence = verify(args.repo, args.run_id, args.revision, args.tag)
    if args.published:
        if not args.tag:
            raise ValueError("published readback requires a tag")
        release = api(args.repo, "releases/tags/" + args.tag)
        latest = api(args.repo, "releases/latest")
        validate_published(release, latest, args.tag)
        evidence["published_latest_stable_verified"] = True
        evidence["release_url"] = release["html_url"]
    if bool(args.expected_assets) != bool(args.downloaded_assets):
        raise ValueError("both asset directories are required")
    if args.expected_assets:
        validate_assets(args.expected_assets, args.downloaded_assets)
        evidence["asset_readback_verified"] = True
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("x", encoding="utf-8") as dest:
        dest.write(json.dumps(evidence, indent=2) + "\n")
    print("Exact-SHA release gate verified:", args.revision)
