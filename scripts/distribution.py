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
DOCS = ("LICENSE", "THIRD_PARTY_NOTICES.md", "QUICKSTART.md", "README.md", "docs/SPEC.md", "docs/PLAN.md", "docs/DEVELOPMENT.md", "docs/CAPABILITIES.md")
TARGETS = ("linux-amd64", "linux-arm64", "windows-amd64", "windows-arm64", "darwin-amd64", "darwin-arm64")
SEMVER = r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?"


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def run(args, **kwargs):
    return subprocess.run(args, capture_output=True, check=True, timeout=120, **kwargs).stdout.decode().strip()


def validate_version(version):
    if version != "dev" and (not isinstance(version, str) or not re.fullmatch(SEMVER, version)):
        raise ValueError("version must be dev or a semantic version without a v prefix")


def package_name(revision, target, version="dev"):
    validate_version(version)
    if not isinstance(revision, str) or not re.fullmatch("[0-9a-f]{40}", revision):
        raise ValueError("invalid source revision")
    if target not in TARGETS:
        raise ValueError("unsupported distribution target: " + str(target))
    return "pockettopo-exporter-" + (revision if version == "dev" else version) + "-" + target + ".zip"


def read_json(raw):
    def unique_keys(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key: " + key)
            result[key] = value
        return result
    result = json.loads(raw, object_pairs_hook=unique_keys)
    if not isinstance(result, dict):
        raise ValueError("expected JSON object")
    return result


def validate_metadata(build, revision, target, version):
    if type(build.get("schema_version")) is not int or build["schema_version"] != 1:
        raise ValueError("unsupported build manifest schema")
    if (build.get("source_revision"), build.get("target"), build.get("cli_version")) != (revision, target, version):
        raise ValueError("package source revision, target or CLI version differs")
    if type(build.get("cgo_enabled")) is not int or build["cgo_enabled"] != 0:
        raise ValueError("package requires CGO_ENABLED=0")
    if not isinstance(build.get("go_version"), str) or not re.fullmatch(
            r"go version go[0-9]+\.[0-9]+\.[0-9]+(?:[a-z]+[0-9]+)? " + re.escape(target.replace("-", "/")), build["go_version"]):
        raise ValueError("package Go toolchain or target differs")


def read_package(path, revision, target, version="dev"):
    if Path(path).name != package_name(revision, target, version):
        raise ValueError("archive filename differs from package identity")
    with zipfile.ZipFile(path) as archive:
        names = archive.namelist()
        if len(set(names)) != len(names):
            raise ValueError("duplicate archive member")
        binary = "pockettopo-exporter" + (".exe" if target.startswith("windows-") else "")
        if set(names) != {*DOCS, binary, "BUILD.json", "SHA256SUMS"}:
            raise ValueError("archive has missing or unexpected members")
        if any(x.file_size > 64 * 1024 * 1024 for x in archive.infolist()):
            raise ValueError("archive payload limit")
        if any(x.create_system != 3 or stat.S_IFMT(x.external_attr >> 16) != stat.S_IFREG for x in archive.infolist()):
            raise ValueError("archive contains nonregular payload")
        if (archive.getinfo(binary).external_attr >> 16) & 0o777 != 0o755:
            raise ValueError("binary executable mode missing")
        payload = {name: archive.read(name) for name in names}
    build = read_json(payload["BUILD.json"])
    validate_metadata(build, revision, target, version)
    hashes = {k: digest(v) for k, v in payload.items() if k not in ("BUILD.json", "SHA256SUMS")}
    if build.get("files") != hashes:
        raise ValueError("package payload differs from build manifest")
    checksums = "".join(digest(v) + "  " + k + "\n" for k, v in sorted(payload.items()) if k != "SHA256SUMS")
    if payload["SHA256SUMS"] != checksums.encode():
        raise ValueError("package checksum manifest differs")
    return payload


def verify_build_info(build_info, metadata):
    lines = build_info.splitlines()
    if not lines or lines[0].rsplit(": ", 1)[-1] != metadata["go_version"].split()[2]:
        raise ValueError("binary embedded Go toolchain differs")
    fields = {}
    for line in lines[1:]:
        match = re.fullmatch(r"\s+build\s+([^=\s]+)=(.*)", line)
        if match:
            key, value = match.groups()
            if key in fields:
                raise ValueError("duplicate binary build setting")
            fields[key] = value
    goos, goarch = metadata["target"].split("-")
    expected = {"GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0",
                "vcs.revision": metadata["source_revision"], "vcs.modified": "false"}
    if any(fields.get(key) != value for key, value in expected.items()):
        raise ValueError("binary embedded source identity, target or CGO differs")


def smoke(payload, target, revision, version="dev"):
    if "-".join(run(["go", "env", "GOHOSTOS", "GOHOSTARCH"]).splitlines()) != target:
        raise ValueError("distribution smoke requires the native target host")
    metadata = read_json(payload["BUILD.json"])
    validate_metadata(metadata, revision, target, version)
    spec = importlib.util.spec_from_file_location("corpus", REPO / "scripts/validate-corpus.py")
    corpus = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(corpus)
    binary_name = "pockettopo-exporter" + (".exe" if target.startswith("windows-") else "")
    with tempfile.TemporaryDirectory(prefix="pockettopo-offline-") as scratch:
        scratch = Path(scratch)
        for name, raw in payload.items():
            dest = scratch / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_bytes(raw)
        binary = scratch / binary_name
        binary.chmod(0o755)
        build_info = run(["go", "version", "-m", str(binary)])
        verify_build_info(build_info, metadata)
        def invoke(args):
            # Absolute executable, empty PATH and unrelated cwd prove no source,
            # Go, Wine, PocketTopo or shell is required by the distributed CLI.
            return subprocess.run([str(binary), *map(str, args)], cwd=scratch,
                                  env={**os.environ, "PATH": ""}, capture_output=True, timeout=30)
        version_result = invoke(["--version"])
        help_result = invoke(["--help"])
        if (version_result.returncode != 0 or version_result.stderr or
                version_result.stdout != ("pockettopo-exporter " + version + "\n").encode() or
                help_result.returncode != 0 or help_result.stderr):
            raise ValueError("distributed CLI help/version failed")
        source = scratch / "inspection.top"
        shutil.copyfile(REPO / "internal/top/testdata/api-drawings.top", source)
        inspection = invoke(["inspect", source, "--json"])
        if inspection.returncode != 0 or inspection.stderr or read_json(inspection.stdout).get("tool_version") != version:
            raise ValueError("distributed inspection version differs")
        cases = json.loads((REPO / "validation/distribution-cases.json").read_text())
        cases += json.loads((REPO / "validation/p15-distribution-cases.json").read_text())
        cases += json.loads((REPO / "validation/p16-distribution-cases.json").read_text())
        for index, case in enumerate(cases):
            source = scratch / Path(case["input"]).name
            shutil.copyfile(REPO / case["input"], source)
            output = scratch / ("export-" + str(index))
            result = invoke(["export", source, "--format", case["format"], "--output", output, *case["options"]])
            record = corpus.export_result(output, result.returncode, result.stderr, case["format"], digest(source.read_bytes()), source.stat().st_size)
            if record["status"] != "supported" or (output / corpus.ARTIFACTS[case["format"]]).read_bytes() != (REPO / case["golden"]).read_bytes():
                raise ValueError("distributed CLI native bytes differ: " + case["format"])
            if read_json((output / "report.json").read_bytes()).get("tool_version") != version:
                raise ValueError("distributed export report version differs: " + case["format"])
            saved = {p.name: p.read_bytes() for p in output.iterdir()}
            again = invoke(["export", source, "--format", case["format"], "--output", output, *case["options"]])
            if again.returncode != 1 or again.stdout or saved != {p.name: p.read_bytes() for p in output.iterdir()}:
                raise ValueError("distributed CLI replaced existing output")
            if case["format"] == "dxf":
                side = scratch / "export-dxf-side"
                result = invoke(["export", source, "--format", "dxf", "--output", side, "--view", "side"])
                if result.returncode != 0 or result.stderr or read_json((side / "report.json").read_bytes()).get("tool_version") != version:
                    raise ValueError("distributed export report version differs: dxf side")
        bad = scratch / "bad.top"
        bad.write_bytes(b"Top\x03")
        for fmt in corpus.ARTIFACTS:
            output = scratch / ("bad-" + fmt)
            result = invoke(["export", bad, "--format", fmt, "--output", output])
            if result.returncode != 1 or result.stdout or not result.stderr or output.exists():
                raise ValueError("distributed CLI malformed-input contract failed")
    return {"schema_version": 1, "source_revision": revision, "target": target,
            "cli_version": version, "go_version": metadata["go_version"], "cgo_enabled": 0,
            "native_runtime": True, "version_verified": True, "inspection_version_verified": True,
            "embedded_identity_verified": True,
            "native_byte_cases": sum(x["format"] not in ("srv", "svx") for x in cases),
            "destination_byte_cases": sum(x["format"] in ("srv", "svx") for x in cases),
            "offline": True, "no_replace": True, "malformed_input": True}


def build(output, version="dev"):
    validate_version(version)
    if run(["git", "status", "--porcelain"], cwd=REPO):
        raise ValueError("distribution requires a clean committed source checkout")
    revision = run(["git", "rev-parse", "HEAD"], cwd=REPO)
    if not re.fullmatch("[0-9a-f]{40}", revision):
        raise ValueError("invalid source revision")
    platforms = run(["go", "env", "GOOS", "GOARCH", "GOHOSTOS", "GOHOSTARCH"], cwd=REPO).splitlines()
    if len(platforms) != 4 or platforms[:2] != platforms[2:]:
        raise ValueError("distribution build requires the native target host")
    target = "-".join(platforms[:2])
    if target not in TARGETS:
        raise ValueError("unsupported distribution target: " + target)
    with tempfile.TemporaryDirectory(prefix="pockettopo-package-") as scratch:
        scratch = Path(scratch)
        binary = "pockettopo-exporter" + (".exe" if target.startswith("windows-") else "")
        run(["go", "build", "-trimpath", "-buildvcs=true", "-ldflags",
             "-X pockettopo-exporter/internal/cli.Version=" + version,
             "-o", str(scratch / binary), "./cmd/pockettopo-exporter"], cwd=REPO,
            env={**os.environ, "CGO_ENABLED": "0"})
        payload = {name: (REPO / name).read_bytes() for name in DOCS}
        payload[binary] = (scratch / binary).read_bytes()
        metadata = {"schema_version": 1, "source_revision": revision, "target": target,
                    "go_version": run(["go", "version"]), "cli_version": version, "cgo_enabled": 0,
                    "files": {name: digest(raw) for name, raw in payload.items()}}
        payload["BUILD.json"] = (json.dumps(metadata, indent=2, sort_keys=True) + "\n").encode()
        payload["SHA256SUMS"] = "".join(digest(v) + "  " + k + "\n" for k, v in sorted(payload.items())).encode()
        name = package_name(revision, target, version)
        archive_path = scratch / name
        with zipfile.ZipFile(archive_path, "x", compression=zipfile.ZIP_DEFLATED) as archive:
            for filename, raw in sorted(payload.items()):
                info = zipfile.ZipInfo(filename, date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (0o100755 if filename == binary else 0o100644) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(info, raw)
        report = smoke(read_package(archive_path, revision, target, version), target, revision, version)
        report.update({"archive": name, "archive_sha256": digest(archive_path.read_bytes())})
        # Reserve only after a complete archive has passed readback and smoke.
        output.mkdir(parents=True, exist_ok=False)
        shutil.copyfile(archive_path, output / name)
        (output / (name + ".sha256")).write_bytes((digest(archive_path.read_bytes()) + "  " + name + "\n").encode())
        (output / "validation.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        print("Distribution verified:", name)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--version", default="dev", help="CLI semantic version, or dev for source-SHA named packages")
    args = parser.parse_args()
    build(args.output, args.version)
