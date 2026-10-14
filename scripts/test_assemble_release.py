"""Fail-closed controls for assembling one six-target native release."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from test_validation import checksum_payload, module, package_payload, write_package


REVISION = "a" * 40
VERSION = "1.0.0"


def write_set(root, target, revision=REVISION, version=VERSION):
    distribution = module("distribution")
    directory = root / ("release-" + target + "-" + revision)
    directory.mkdir(parents=True, exist_ok=True)
    archive = directory / ("pockettopo-exporter-" + version + "-" + target + ".zip")
    payload = package_payload(distribution, revision, target, version)
    write_package(archive, payload)
    archive_hash = hashlib.sha256(archive.read_bytes()).hexdigest()
    (directory / (archive.name + ".sha256")).write_bytes((archive_hash + "  " + archive.name + "\n").encode())
    metadata = json.loads(payload["BUILD.json"])
    report = {"schema_version": 1, "archive": archive.name, "archive_sha256": archive_hash,
              "source_revision": revision, "target": target, "cli_version": version,
              "go_version": metadata["go_version"], "cgo_enabled": 0, "native_runtime": True,
              "version_verified": True, "inspection_version_verified": True,
              "embedded_identity_verified": True, "native_byte_cases": 6,
              "destination_byte_cases": 2, "offline": True, "no_replace": True, "malformed_input": True}
    (directory / "validation.json").write_text(json.dumps(report), encoding="utf-8")
    return directory, archive, payload, report


class ReleaseAssemblyControls(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory()
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.inputs = self.root / "downloaded"
        self.output = self.root / "release"
        self.distribution = module("distribution")
        self.sets = {target: write_set(self.inputs, target) for target in self.distribution.TARGETS}

    def assemble(self):
        return module("assemble-release").assemble(self.inputs, self.output, REVISION, VERSION)

    def reject(self):
        with self.assertRaises(ValueError):
            self.assemble()
        self.assertFalse(self.output.exists(), "failed validation must not reserve release output")

    def replace_archive(self, target, payload):
        directory, archive, _, report = self.sets[target]
        write_package(archive, checksum_payload(payload))
        archive_hash = hashlib.sha256(archive.read_bytes()).hexdigest()
        (directory / (archive.name + ".sha256")).write_bytes((archive_hash + "  " + archive.name + "\n").encode())
        report["archive_sha256"] = archive_hash
        (directory / "validation.json").write_text(json.dumps(report), encoding="utf-8")

    def test_complete_release_copies_exact_archive_bytes_and_records_native_evidence(self):
        before = {p.relative_to(self.inputs): p.read_bytes() for p in self.inputs.rglob("*") if p.is_file()}
        self.assemble()
        archives = sorted(self.output.glob("*.zip"))
        self.assertEqual(len(archives), 6)
        self.assertEqual({p.name for p in self.output.iterdir()},
                         {p.name for p in archives} | {"SHA256SUMS", "release-validation.json"})
        for _, original, _, _ in self.sets.values():
            self.assertEqual((self.output / original.name).read_bytes(), original.read_bytes())
        expected = "".join(hashlib.sha256(p.read_bytes()).hexdigest() + "  " + p.name + "\n" for p in archives)
        self.assertEqual((self.output / "SHA256SUMS").read_bytes(), expected.encode())
        report = json.loads((self.output / "release-validation.json").read_text())
        self.assertEqual((report["schema_version"], report["source_revision"], report["cli_version"]),
                         (1, REVISION, VERSION))
        self.assertEqual({p["target"] for p in report["packages"]}, set(self.distribution.TARGETS))
        self.assertTrue(all(p["native_runtime"] is True for p in report["packages"]))
        self.assertEqual(before, {p.relative_to(self.inputs): p.read_bytes()
                                 for p in self.inputs.rglob("*") if p.is_file()})

    def test_missing_duplicate_foreign_and_renamed_archives_are_rejected(self):
        target = "linux-arm64"
        directory, archive, _, _ = self.sets[target]
        archive.unlink()
        self.reject()
        shutil.rmtree(directory)
        self.sets[target] = write_set(self.inputs, target)
        duplicate = self.inputs / "another-artifact"
        shutil.copytree(self.sets[target][0], duplicate)
        self.reject()
        shutil.rmtree(duplicate)
        archive = self.sets[target][1]
        archive.rename(archive.with_name("pockettopo-exporter-1.0.0-freebsd-arm64.zip"))
        self.reject()

    def test_missing_checksum_missing_validation_and_extra_assets_are_rejected(self):
        directory, archive, _, _ = self.sets["windows-arm64"]
        sidecar = directory / (archive.name + ".sha256")
        raw = sidecar.read_bytes()
        sidecar.unlink()
        self.reject()
        sidecar.write_bytes(raw)
        validation = directory / "validation.json"
        raw = validation.read_bytes()
        validation.unlink()
        self.reject()
        validation.write_bytes(raw)
        (directory / "unexpected.zip").write_bytes(b"foreign")
        self.reject()

    def test_foreign_revision_and_version_even_with_consistent_hashes_are_rejected(self):
        target = "darwin-arm64"
        shutil.rmtree(self.sets[target][0])
        foreign = write_set(self.inputs, target, revision="b" * 40)
        self.reject()
        shutil.rmtree(foreign[0])
        write_set(self.inputs, target, version="1.0.1")
        self.reject()

    def test_changed_archive_payload_build_manifest_and_internal_checksums_are_rejected(self):
        target = "linux-amd64"
        _, _, payload, _ = self.sets[target]
        metadata = json.loads(payload["BUILD.json"])
        for key, value in [("source_revision", "b" * 40), ("cli_version", "dev"),
                           ("cgo_enabled", 1), ("target", "windows-amd64"), ("files", {})]:
            with self.subTest(key=key):
                self.replace_archive(target, dict(payload, **{"BUILD.json": json.dumps(dict(metadata, **{key: value})).encode()}))
                self.reject()
        self.replace_archive(target, dict(payload, **{"README.md": b"changed"}))
        self.reject()
        directory, archive, _, report = self.sets[target]
        write_package(archive, dict(payload, SHA256SUMS=b"0" * 64))
        archive_hash = hashlib.sha256(archive.read_bytes()).hexdigest()
        (directory / (archive.name + ".sha256")).write_bytes((archive_hash + "  " + archive.name + "\n").encode())
        report["archive_sha256"] = archive_hash
        (directory / "validation.json").write_text(json.dumps(report), encoding="utf-8")
        self.reject()

    def test_sidecar_must_match_archive_bytes_and_exact_filename(self):
        directory, archive, _, _ = self.sets["windows-amd64"]
        sidecar = directory / (archive.name + ".sha256")
        correct = hashlib.sha256(archive.read_bytes()).hexdigest()
        for content in ("0" * 64 + "  " + archive.name + "\n", correct + "  foreign.zip\n",
                        correct + "  " + archive.name + "\n" + correct + "  duplicate.zip\n"):
            with self.subTest(content=content):
                sidecar.write_bytes(content.encode())
                self.reject()

    def test_native_report_requires_every_control_exact_counts_and_matching_identity(self):
        directory, _, _, report = self.sets["darwin-amd64"]
        changes = [{key: False} for key in ("native_runtime", "version_verified", "inspection_version_verified",
                   "embedded_identity_verified", "offline", "no_replace", "malformed_input")]
        changes += [{"source_revision": "b" * 40}, {"target": "darwin-arm64"}, {"cli_version": "dev"},
                    {"native_byte_cases": 5}, {"native_byte_cases": True}, {"destination_byte_cases": 1},
                    {"archive_sha256": "b" * 64}, {"cgo_enabled": 1}, {"go_version": "go version go1.26.2 darwin/amd64"}]
        for change in changes:
            with self.subTest(change=change):
                (directory / "validation.json").write_text(json.dumps(dict(report, **change)), encoding="utf-8")
                self.reject()
        for key in report:
            with self.subTest(missing=key):
                (directory / "validation.json").write_text(json.dumps({k: v for k, v in report.items() if k != key}), encoding="utf-8")
                self.reject()

    def test_existing_output_is_preserved_and_dev_or_invalid_revision_cannot_publish(self):
        self.output.mkdir()
        marker = self.output / "existing"
        marker.write_bytes(b"preserve")
        with self.assertRaises(FileExistsError):
            self.assemble()
        self.assertEqual(marker.read_bytes(), b"preserve")
        marker.unlink()
        self.output.rmdir()
        assembler = module("assemble-release")
        for revision, version in [("a" * 39, VERSION), (REVISION, "dev"), (REVISION, "v1.0.0")]:
            with self.subTest(revision=revision, version=version), self.assertRaises(ValueError):
                assembler.assemble(self.inputs, self.output, revision, version)
        self.assertFalse(self.output.exists())

    def test_cli_runs_offline_without_modifying_input(self):
        result = subprocess.run([sys.executable, str(Path(__file__).with_name("assemble-release.py")),
                                 "--input", str(self.inputs), "--output", str(self.output),
                                 "--revision", REVISION, "--version", VERSION], capture_output=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr.decode())
        self.assertTrue((self.output / "SHA256SUMS").is_file())

    def test_assembled_archive_readback_rejects_copy_corruption_and_removes_partial_output(self):
        assembler = module("assemble-release")
        copyfile = shutil.copyfile
        def damaged_copy(source, destination):
            copyfile(source, destination)
            destination.write_bytes(b"damaged during copy")
        with patch.object(assembler.shutil, "copyfile", side_effect=damaged_copy):
            with self.assertRaisesRegex(ValueError, "assembled archive"):
                assembler.assemble(self.inputs, self.output, REVISION, VERSION)
        self.assertFalse(self.output.exists())

    def test_duplicate_validation_keys_and_mixed_toolchains_are_rejected(self):
        target = "darwin-arm64"
        directory, _, payload, report = self.sets[target]
        validation = directory / "validation.json"
        validation.write_bytes(b'{"native_runtime": false, ' + json.dumps(report).encode()[1:])
        self.reject()
        validation.write_text(json.dumps(report), encoding="utf-8")
        metadata = json.loads(payload["BUILD.json"])
        metadata["go_version"] = "go version go1.26.2 darwin/arm64"
        report["go_version"] = metadata["go_version"]
        self.replace_archive(target, dict(payload, **{"BUILD.json": json.dumps(metadata).encode()}))
        self.reject()


if __name__ == "__main__":
    unittest.main()
