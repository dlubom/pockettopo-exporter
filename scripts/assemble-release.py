#!/usr/bin/env python3
"""Verify and assemble six native distribution artifacts from one exact revision."""
import argparse
import importlib.util
import json
from pathlib import Path
import shutil


spec = importlib.util.spec_from_file_location("distribution", Path(__file__).with_name("distribution.py"))
distribution = importlib.util.module_from_spec(spec)
spec.loader.exec_module(distribution)


def verify_native_report(report, metadata, archive_name, archive_hash):
    expected = {"schema_version": 1, "source_revision": metadata["source_revision"],
                "target": metadata["target"], "cli_version": metadata["cli_version"],
                "go_version": metadata["go_version"], "cgo_enabled": 0,
                "archive": archive_name, "archive_sha256": archive_hash,
                "native_byte_cases": 6, "destination_byte_cases": 2,
                "native_runtime": True, "version_verified": True, "inspection_version_verified": True,
                "embedded_identity_verified": True, "offline": True, "no_replace": True,
                "malformed_input": True}
    if any(type(report.get(key)) is not type(value) or report[key] != value for key, value in expected.items()):
        raise ValueError("native distribution evidence is incomplete or differs from archive: " + archive_name)


def assemble(inputs, output, revision, version):
    distribution.validate_version(version)
    if version == "dev":
        raise ValueError("release assembly requires a semantic version")
    names = {distribution.package_name(revision, target, version): target for target in distribution.TARGETS}
    if not inputs.is_dir() or inputs.is_symlink():
        raise ValueError("release input must be a regular artifact directory")
    files = []
    for path in inputs.rglob("*"):
        if path.is_symlink() or not (path.is_file() or path.is_dir()):
            raise ValueError("release input contains a nonregular artifact")
        if path.is_file():
            files.append(path)
    archives = [path for path in files if path.suffix == ".zip"]
    if len(archives) != len(names) or {path.name for path in archives} != set(names):
        raise ValueError("release requires exactly one archive for every supported target")
    consumed = set()
    packages = []
    go_versions = set()
    for archive in sorted(archives, key=lambda p: p.name):
        target = names[archive.name]
        checksum = archive.with_name(archive.name + ".sha256")
        validation = archive.with_name("validation.json")
        if checksum not in files or validation not in files or validation in consumed:
            raise ValueError("archive is missing its unique checksum or native validation: " + archive.name)
        archive_hash = distribution.digest(archive.read_bytes())
        if checksum.read_bytes() != (archive_hash + "  " + archive.name + "\n").encode():
            raise ValueError("archive sidecar checksum or filename differs: " + archive.name)
        payload = distribution.read_package(archive, revision, target, version)
        metadata = distribution.read_json(payload["BUILD.json"])
        report = distribution.read_json(validation.read_bytes())
        verify_native_report(report, metadata, archive.name, archive_hash)
        go_versions.add(metadata["go_version"].split()[2])
        packages.append(report)
        consumed.update((archive, checksum, validation))
    if consumed != set(files):
        raise ValueError("release input contains unexpected or duplicate artifact files")
    if len(go_versions) != 1:
        raise ValueError("release packages use different Go toolchains")
    # Validation above is read-only. Reserve a new directory only for a complete
    # release, preserving both downloaded inputs and any existing release output.
    output.mkdir(parents=True, exist_ok=False)
    try:
        archive_hashes = {item["archive"]: item["archive_sha256"] for item in packages}
        for archive in archives:
            destination = output / archive.name
            shutil.copyfile(archive, destination)
            if distribution.digest(destination.read_bytes()) != archive_hashes[archive.name]:
                raise ValueError("assembled archive readback differs: " + archive.name)
        checksums = "".join(item["archive_sha256"] + "  " + item["archive"] + "\n" for item in packages)
        (output / "SHA256SUMS").write_bytes(checksums.encode())
        result = {"schema_version": 1, "source_revision": revision, "cli_version": version,
                  "go_toolchain": next(iter(go_versions)), "packages": packages}
        (output / "release-validation.json").write_bytes((json.dumps(result, indent=2, sort_keys=True) + "\n").encode())
    except BaseException:
        shutil.rmtree(output)
        raise
    print("Release artifacts verified:", version, revision)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True, type=Path, help="root of downloaded native artifact directories")
    parser.add_argument("--output", required=True, type=Path, help="new release asset directory")
    parser.add_argument("--revision", required=True, help="exact 40-character source SHA")
    parser.add_argument("--version", required=True, help="semantic release version without a v prefix")
    args = parser.parse_args()
    assemble(args.input, args.output, args.revision, args.version)
