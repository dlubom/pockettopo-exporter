#!/usr/bin/env python3
"""Build, checksum and test an offline native-platform distribution from clean Git."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import tempfile
import zipfile

REPO = Path(__file__).resolve().parent.parent
DOCS = ("README.md", "docs/SPEC.md", "docs/PLAN.md", "docs/DEVELOPMENT.md", "docs/CAPABILITIES.md")
TARGETS = ("linux-amd64", "windows-amd64", "darwin-arm64")


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def run(args, **kwargs):
    return subprocess.run(args, capture_output=True, check=True, timeout=120, **kwargs).stdout.decode().strip()


def read_package(path, revision, target):
    with zipfile.ZipFile(path) as archive:
        names = archive.namelist()
        if len(set(names)) != len(names):
            raise ValueError("duplicate archive member")
        binary = "pockettopo-exporter" + (".exe" if target == "windows-amd64" else "")
        if set(names) != {*DOCS, binary, "BUILD.json", "SHA256SUMS"}:
            raise ValueError("archive has missing or unexpected members")
        if any(x.file_size > 64 * 1024 * 1024 for x in archive.infolist()):
            raise ValueError("archive payload limit")
        if any(x.create_system != 3 or stat.S_IFMT(x.external_attr >> 16) != stat.S_IFREG for x in archive.infolist()):
            raise ValueError("archive contains nonregular payload")
        if (archive.getinfo(binary).external_attr >> 16) & 0o777 != 0o755:
            raise ValueError("binary executable mode missing")
        payload = {name: archive.read(name) for name in names}
    build = json.loads(payload["BUILD.json"])
    if (build["schema_version"], build["source_revision"], build["target"]) != (1, revision, target):
        raise ValueError("package source revision or target differs")
    hashes = {k: digest(v) for k, v in payload.items() if k not in ("BUILD.json", "SHA256SUMS")}
    if build["files"] != hashes:
        raise ValueError("package payload differs from build manifest")
    checksums = "".join(digest(v) + "  " + k + "\n" for k, v in sorted(payload.items()) if k != "SHA256SUMS")
    if payload["SHA256SUMS"] != checksums.encode():
        raise ValueError("package checksum manifest differs")
    return payload


def smoke(payload, target, revision):
    spec = importlib.util.spec_from_file_location("corpus", REPO / "scripts/validate-corpus.py")
    corpus = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(corpus)
    binary_name = "pockettopo-exporter" + (".exe" if target == "windows-amd64" else "")
    with tempfile.TemporaryDirectory(prefix="pockettopo-offline-") as scratch:
        scratch = Path(scratch)
        for name, raw in payload.items():
            dest = scratch / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_bytes(raw)
        binary = scratch / binary_name
        binary.chmod(0o755)
        build_info = run(["go", "version", "-m", str(binary)])
        if "vcs.revision=" + revision not in build_info or "vcs.modified=false" not in build_info:
            raise ValueError("binary embedded source identity differs")
        def invoke(args):
            # Absolute executable, empty PATH and unrelated cwd prove no source,
            # Go, Wine, PocketTopo or shell is required by the distributed CLI.
            return subprocess.run([str(binary), *map(str, args)], cwd=scratch,
                                  env={**os.environ, "PATH": ""}, capture_output=True, timeout=30)
        if invoke(["--version"]).stdout != b"pockettopo-exporter dev\n" or invoke(["--help"]).returncode != 0:
            raise ValueError("distributed CLI help/version failed")
        cases = json.loads((REPO / "validation/distribution-cases.json").read_text())
        for index, case in enumerate(cases):
            source = scratch / Path(case["input"]).name
            shutil.copyfile(REPO / case["input"], source)
            output = scratch / ("export-" + str(index))
            result = invoke(["export", source, "--format", case["format"], "--output", output, *case["options"]])
            record = corpus.export_result(output, result.returncode, result.stderr, case["format"], digest(source.read_bytes()), source.stat().st_size)
            if record["status"] != "supported" or (output / corpus.ARTIFACTS[case["format"]]).read_bytes() != (REPO / case["golden"]).read_bytes():
                raise ValueError("distributed CLI native bytes differ: " + case["format"])
            saved = {p.name: p.read_bytes() for p in output.iterdir()}
            again = invoke(["export", source, "--format", case["format"], "--output", output, *case["options"]])
            if again.returncode != 1 or again.stdout or saved != {p.name: p.read_bytes() for p in output.iterdir()}:
                raise ValueError("distributed CLI replaced existing output")
        bad = scratch / "bad.top"
        bad.write_bytes(b"Top\x03")
        for fmt in corpus.ARTIFACTS:
            output = scratch / ("bad-" + fmt)
            result = invoke(["export", bad, "--format", fmt, "--output", output])
            if result.returncode != 1 or result.stdout or not result.stderr or output.exists():
                raise ValueError("distributed CLI malformed-input contract failed")
    return {"source_revision": revision, "target": target, "native_byte_cases": len(cases),
            "offline": True, "no_replace": True, "malformed_input": True}


def build(output):
    if run(["git", "status", "--porcelain"], cwd=REPO):
        raise ValueError("distribution requires a clean committed source checkout")
    revision = run(["git", "rev-parse", "HEAD"], cwd=REPO)
    if not re.fullmatch("[0-9a-f]{40}", revision):
        raise ValueError("invalid source revision")
    target = "-".join(run(["go", "env", "GOOS", "GOARCH"], cwd=REPO).splitlines())
    if target not in TARGETS:
        raise ValueError("unsupported distribution target: " + target)
    with tempfile.TemporaryDirectory(prefix="pockettopo-package-") as scratch:
        scratch = Path(scratch)
        binary = "pockettopo-exporter" + (".exe" if target == "windows-amd64" else "")
        run(["go", "build", "-trimpath", "-buildvcs=true", "-o", str(scratch / binary), "./cmd/pockettopo-exporter"], cwd=REPO)
        payload = {name: (REPO / name).read_bytes() for name in DOCS}
        payload[binary] = (scratch / binary).read_bytes()
        metadata = {"schema_version": 1, "source_revision": revision, "target": target,
                    "go_version": run(["go", "version"]), "cli_version": "dev",
                    "files": {name: digest(raw) for name, raw in payload.items()}}
        payload["BUILD.json"] = (json.dumps(metadata, indent=2, sort_keys=True) + "\n").encode()
        payload["SHA256SUMS"] = "".join(digest(v) + "  " + k + "\n" for k, v in sorted(payload.items())).encode()
        name = "pockettopo-exporter-" + revision + "-" + target + ".zip"
        archive_path = scratch / name
        with zipfile.ZipFile(archive_path, "x", compression=zipfile.ZIP_DEFLATED) as archive:
            for filename, raw in sorted(payload.items()):
                info = zipfile.ZipInfo(filename, date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (0o100755 if filename == binary else 0o100644) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(info, raw)
        report = smoke(read_package(archive_path, revision, target), target, revision)
        # Reserve only after a complete archive has passed readback and smoke.
        output.mkdir(parents=True, exist_ok=False)
        shutil.copyfile(archive_path, output / name)
        (output / (name + ".sha256")).write_text(digest(archive_path.read_bytes()) + "  " + name + "\n", encoding="utf-8")
        (output / "validation.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        print("Distribution verified:", name)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    build(parser.parse_args().output)
