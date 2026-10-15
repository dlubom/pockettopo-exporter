"""Negative controls for P14 evidence and distribution integrity."""
import hashlib
import importlib.util
import json
import stat
import subprocess
from pathlib import Path
import tempfile
import unittest
import zipfile
import warnings
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + ".py"))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


def package_payload(distribution, revision, target, version="dev"):
    payload = {name: b"docs" for name in distribution.DOCS}
    binary = "pockettopo-exporter" + (".exe" if target.startswith("windows-") else "")
    payload[binary] = b"binary"
    metadata = {"schema_version": 1, "source_revision": revision,
                "target": target, "cli_version": version, "cgo_enabled": 0,
                "go_version": "go version go1.26.3 " + target.replace("-", "/"),
                "files": {k: hashlib.sha256(v).hexdigest() for k, v in payload.items()}}
    payload["BUILD.json"] = json.dumps(metadata).encode()
    return checksum_payload(payload)


def checksum_payload(payload):
    payload = {k: v for k, v in payload.items() if k != "SHA256SUMS"}
    payload["SHA256SUMS"] = "".join(hashlib.sha256(v).hexdigest() + "  " + k + "\n"
                                      for k, v in sorted(payload.items())).encode()
    return payload


def write_package(path, payload):
    with zipfile.ZipFile(path, "w") as archive:
        for name, raw in sorted(payload.items()):
            info = zipfile.ZipInfo(name)
            info.create_system = 3
            info.external_attr = (0o100755 if name.startswith("pockettopo-exporter") else 0o100644) << 16
            archive.writestr(info, raw)


class CorpusControls(unittest.TestCase):
    def test_inventory_checks_paths_bytes_and_git_identity(self):
        # Removing, adding, or changing an input must invalidate the corpus.
        corpus = module("validate-corpus")
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            raw = b"Top\x03"
            item = {"path": "a.top", "size_bytes": 4,
                    "sha256": hashlib.sha256(raw).hexdigest(),
                    "git_blob": hashlib.sha1(b"blob 4\0" + raw).hexdigest()}
            (root / "a.top").write_bytes(raw)
            corpus.verify_inputs(root, [item])
            (root / "extra.TOP").write_bytes(raw)
            with self.assertRaises(ValueError):
                corpus.verify_inputs(root, [item])
            (root / "extra.TOP").unlink()
            (root / "a.top").write_bytes(b"Top\x02")
            with self.assertRaises(ValueError):
                corpus.verify_inputs(root, [item])
            (root / "a.top").unlink()
            with self.assertRaises(ValueError):
                corpus.verify_inputs(root, [item])
            (root / "a.top").write_bytes(raw)
            with self.assertRaises(ValueError):
                corpus.verify_inputs(root, [item, item])
            with self.assertRaises(ValueError):
                corpus.verify_inputs(root, [dict(item, git_blob="0" * 40)])

    def test_export_requires_hashes_complete_files_and_failure_cleanup(self):
        corpus = module("validate-corpus")
        with tempfile.TemporaryDirectory() as d:
            out = Path(d) / "export"
            out.mkdir()
            artifact = b"native\r\n"
            report = {"schema_version": 1, "source_sha256": "a" * 64,
                      "source_size_bytes": 4,
                      "text_sha256": hashlib.sha256(artifact).hexdigest(),
                      "text_size_bytes": len(artifact)}
            (out / "survey.txt").write_bytes(artifact)
            (out / "report.json").write_text(json.dumps(report))
            corpus.export_result(out, 0, b"", "text", "a" * 64, 4)
            # P09/P11/P12 retain source size in the current input inventory.
            report.pop("source_size_bytes")
            report["inputs"] = [{"role": "current", "sha256": "a" * 64, "size_bytes": 4}]
            (out / "report.json").write_text(json.dumps(report))
            corpus.export_result(out, 0, b"", "therion", "a" * 64, 4)
            (out / "survey.txt").write_bytes(b"tampered")
            with self.assertRaises(ValueError):
                corpus.export_result(out, 0, b"", "text", "a" * 64, 4)
            with self.assertRaises(ValueError):
                corpus.export_result(out, 1, b"Cannot export: resource_limit", "text", "a" * 64, 4)
            (out / "survey.txt").unlink()
            with self.assertRaises(ValueError):
                corpus.export_result(out, 0, b"", "text", "a" * 64, 4)
            (out / "report.json").unlink()
            out.rmdir()
            with self.assertRaises(ValueError):
                corpus.export_result(out, 1, b"unclassified error", "text", "a" * 64, 4)


class DistributionControls(unittest.TestCase):
    def test_archive_rejects_tampering_missing_extra_and_duplicate_members(self):
        distribution = module("distribution")
        with tempfile.TemporaryDirectory() as d:
            archive = Path(d) / ("pockettopo-exporter-" + "a" * 40 + "-linux-amd64.zip")
            payload = package_payload(distribution, "a" * 40, "linux-amd64")
            def write(values, duplicate=False, foreign_type=None):
                with zipfile.ZipFile(archive, "w") as z:
                    for name, raw in values.items():
                        info = zipfile.ZipInfo(name)
                        info.external_attr = (0o100755 if name == "pockettopo-exporter" else 0o100644) << 16
                        info.create_system = 3
                        if foreign_type and name == foreign_type[0]:
                            info.external_attr = (foreign_type[1] | 0o755) << 16
                        z.writestr(info, raw)
                    if duplicate:
                        with warnings.catch_warnings():
                            warnings.simplefilter("ignore", UserWarning)
                            z.writestr("README.md", b"foreign")
            write(payload)
            distribution.read_package(archive, "a" * 40, "linux-amd64")
            for damaged in [dict(payload, **{"README.md": b"changed"}),
                            {k: v for k, v in payload.items() if k != "README.md"},
                            dict(payload, **{"../foreign": b"extra"})]:
                write(damaged)
                with self.assertRaises(ValueError):
                    distribution.read_package(archive, "a" * 40, "linux-amd64")
            write(payload, duplicate=True)
            with self.assertRaises(ValueError):
                distribution.read_package(archive, "a" * 40, "linux-amd64")
            write(payload)
            with self.assertRaises(ValueError):
                distribution.read_package(archive, "b" * 40, "linux-amd64")
            with self.assertRaises(ValueError):
                distribution.read_package(archive, "a" * 40, "windows-amd64")
            for name, mode in [("pockettopo-exporter", stat.S_IFLNK),
                               ("README.md", stat.S_IFLNK), ("README.md", stat.S_IFDIR)]:
                write(payload, foreign_type=(name, mode))
                with self.assertRaises(ValueError):
                    distribution.read_package(archive, "a" * 40, "linux-amd64")

    def test_six_targets_required_documentation_and_release_identity(self):
        distribution = module("distribution")
        targets = {"linux-amd64", "linux-arm64", "windows-amd64", "windows-arm64",
                   "darwin-amd64", "darwin-arm64"}
        self.assertEqual(set(distribution.TARGETS), targets)
        self.assertEqual(set(distribution.DOCS), {"LICENSE", "THIRD_PARTY_NOTICES.md", "QUICKSTART.md", "README.md",
                         "docs/SPEC.md", "docs/PLAN.md", "docs/DEVELOPMENT.md", "docs/CAPABILITIES.md"})
        with tempfile.TemporaryDirectory() as d:
            for target in sorted(targets):
                path = Path(d) / ("pockettopo-exporter-1.0.0-" + target + ".zip")
                payload = package_payload(distribution, "a" * 40, target, "1.0.0")
                write_package(path, payload)
                self.assertEqual(distribution.read_package(path, "a" * 40, target, "1.0.0"), payload)
                with self.assertRaises(ValueError):
                    distribution.read_package(path, "a" * 40, target)
                wrong_name = path.with_name("foreign-" + target + ".zip")
                path.rename(wrong_name)
                with self.assertRaises(ValueError):
                    distribution.read_package(wrong_name, "a" * 40, target, "1.0.0")

    def test_build_metadata_rejects_changed_or_missing_version_toolchain_and_cgo(self):
        distribution = module("distribution")
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "pockettopo-exporter-1.0.0-linux-amd64.zip"
            payload = package_payload(distribution, "a" * 40, "linux-amd64", "1.0.0")
            metadata = json.loads(payload["BUILD.json"])
            changes = [{"cli_version": "dev"}, {"source_revision": "b" * 40},
                       {"target": "linux-arm64"}, {"cgo_enabled": 1},
                       {"cgo_enabled": False}, {"go_version": "unknown"},
                       {"go_version": "go version go1.26.3 windows/amd64"},
                       {"files": {}}]
            for change in changes:
                with self.subTest(change=change):
                    altered = dict(payload, **{"BUILD.json": json.dumps(dict(metadata, **change)).encode()})
                    write_package(path, checksum_payload(altered))
                    with self.assertRaises(ValueError):
                        distribution.read_package(path, "a" * 40, "linux-amd64", "1.0.0")
            for key in ("cli_version", "go_version", "cgo_enabled", "source_revision", "files"):
                altered = {k: v for k, v in metadata.items() if k != key}
                write_package(path, checksum_payload(dict(payload, **{"BUILD.json": json.dumps(altered).encode()})))
                with self.assertRaises(ValueError):
                    distribution.read_package(path, "a" * 40, "linux-amd64", "1.0.0")

    def test_embedded_binary_identity_checks_target_revision_cgo_and_toolchain(self):
        distribution = module("distribution")
        metadata = json.loads(package_payload(distribution, "a" * 40, "windows-arm64", "1.0.0")["BUILD.json"])
        build_info = "binary.exe: go1.26.3\n" + "\n".join("\tbuild\t" + x for x in [
            "GOOS=windows", "GOARCH=arm64", "CGO_ENABLED=0", "vcs.revision=" + "a" * 40,
            "vcs.modified=false"])
        distribution.verify_build_info(build_info, metadata)
        for old, new in [("GOOS=windows", "GOOS=linux"), ("GOARCH=arm64", "GOARCH=amd64"),
                         ("CGO_ENABLED=0", "CGO_ENABLED=1"), ("vcs.modified=false", "vcs.modified=true"),
                         ("vcs.revision=" + "a" * 40, "vcs.revision=" + "b" * 40),
                         ("go1.26.3", "go1.26.2")]:
            with self.subTest(new=new), self.assertRaises(ValueError):
                distribution.verify_build_info(build_info.replace(old, new), metadata)
        with self.assertRaises(ValueError):
            distribution.verify_build_info(build_info.replace("\tbuild\tCGO_ENABLED=0", ""), metadata)

    def test_native_smoke_rejects_wrong_host_cli_and_inspection_versions(self):
        distribution = module("distribution")
        payload = package_payload(distribution, "a" * 40, "linux-amd64", "1.0.0")
        build_info = "binary: go1.26.3\n" + "\n".join("\tbuild\t" + x for x in [
            "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "vcs.revision=" + "a" * 40, "vcs.modified=false"])
        with patch.object(distribution, "run", return_value="linux\narm64"), self.assertRaisesRegex(ValueError, "native target"):
            distribution.smoke(payload, "linux-amd64", "a" * 40, "1.0.0")
        for cli_version, inspection_version in [("dev", "1.0.0"), ("1.0.0", "dev")]:
            with self.subTest(cli_version=cli_version, inspection_version=inspection_version):
                def invoke(args, **kwargs):
                    if args[1:] == ["--version"]:
                        return subprocess.CompletedProcess(args, 0, ("pockettopo-exporter " + cli_version + "\n").encode(), b"")
                    if args[1:] == ["--help"]:
                        return subprocess.CompletedProcess(args, 0, b"help", b"")
                    if args[1] == "inspect":
                        return subprocess.CompletedProcess(args, 0, json.dumps({"tool_version": inspection_version}).encode(), b"")
                    self.fail("version mismatch should stop before export")
                with patch.object(distribution, "run", side_effect=["linux\namd64", build_info]), \
                     patch.object(distribution.subprocess, "run", side_effect=invoke), self.assertRaises(ValueError):
                    distribution.smoke(payload, "linux-amd64", "a" * 40, "1.0.0")

    def test_manifest_duplicate_keys_and_binary_without_execute_mode_are_rejected(self):
        distribution = module("distribution")
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "pockettopo-exporter-1.0.0-linux-amd64.zip"
            payload = package_payload(distribution, "a" * 40, "linux-amd64", "1.0.0")
            duplicate = b'{"cli_version": "dev", ' + payload["BUILD.json"][1:]
            write_package(path, checksum_payload(dict(payload, **{"BUILD.json": duplicate})))
            with self.assertRaisesRegex(ValueError, "duplicate JSON"):
                distribution.read_package(path, "a" * 40, "linux-amd64", "1.0.0")
            with zipfile.ZipFile(path, "w") as archive:
                for name, raw in payload.items():
                    info = zipfile.ZipInfo(name)
                    info.create_system = 3
                    info.external_attr = 0o100644 << 16
                    archive.writestr(info, raw)
            with self.assertRaisesRegex(ValueError, "executable mode"):
                distribution.read_package(path, "a" * 40, "linux-amd64", "1.0.0")

    def test_native_smoke_checks_every_export_report_version(self):
        distribution = module("distribution")
        corpus = module("validate-corpus")
        cases = []
        for filename in ("distribution-cases.json", "p15-distribution-cases.json", "p16-distribution-cases.json"):
            cases += json.loads((distribution.REPO / "validation" / filename).read_text())
        cases.append({"format": "dxf", "golden": "internal/exportdxf/testdata/features-o1.dxf"})
        self.assertEqual(len(cases), 9)
        build_info = "binary: go1.26.3\n" + "\n".join("\tbuild\t" + x for x in [
            "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "vcs.revision=" + "a" * 40, "vcs.modified=false"])

        def smoke(version, bad_index=None, bad_version=None):
            def invoke(args, **kwargs):
                if args[1:] == ["--version"]:
                    return subprocess.CompletedProcess(args, 0, ("pockettopo-exporter " + version + "\n").encode(), b"")
                if args[1:] == ["--help"]:
                    return subprocess.CompletedProcess(args, 0, b"help", b"")
                if args[1] == "inspect":
                    return subprocess.CompletedProcess(args, 0, json.dumps({"tool_version": version}).encode(), b"")
                self.assertEqual(args[1], "export")
                self.assertEqual(args[3:6:2], ["--format", "--output"])
                output = Path(args[args.index("--output") + 1])
                if output.exists() or Path(args[2]).name == "bad.top":
                    return subprocess.CompletedProcess(args, 1, b"", b"Cannot export: rejected")
                index = len(cases) - 1 if output.name == "export-dxf-side" else int(output.name.removeprefix("export-"))
                case = cases[index]
                raw = (distribution.REPO / case["golden"]).read_bytes()
                source = Path(args[2]).read_bytes()
                prefix = corpus.HASH_FIELDS[case["format"]]
                report = {"schema_version": 1, "source_sha256": hashlib.sha256(source).hexdigest(),
                          "source_size_bytes": len(source), prefix + "_sha256": hashlib.sha256(raw).hexdigest(),
                          prefix + "_size_bytes": len(raw)}
                tool_version = bad_version if index == bad_index else version
                if tool_version is not None:
                    report["tool_version"] = tool_version
                output.mkdir()
                (output / corpus.ARTIFACTS[case["format"]]).write_bytes(raw)
                (output / "report.json").write_text(json.dumps(report))
                return subprocess.CompletedProcess(args, 0, b"Exported", b"")

            payload = package_payload(distribution, "a" * 40, "linux-amd64", version)
            with patch.object(distribution, "run", side_effect=["linux\namd64", build_info]), \
                 patch.object(distribution.subprocess, "run", side_effect=invoke):
                return distribution.smoke(payload, "linux-amd64", "a" * 40, version)

        for version in ("dev", "1.0.0", "2.3.4-rc.1+issue2"):
            with self.subTest(version=version):
                self.assertEqual(smoke(version)["cli_version"], version)
        for index, case in enumerate(cases):
            for wrong in (None, "dev", "0.9.0"):
                with self.subTest(profile=case, wrong=wrong), self.assertRaisesRegex(ValueError, "export report version"):
                    smoke("1.0.0", index, wrong)

    def test_builder_rejects_dirty_checkout_and_cross_target_before_build(self):
        distribution = module("distribution")
        with patch.object(distribution, "run", return_value=" M changed.go") as run, self.assertRaisesRegex(ValueError, "clean committed"):
            distribution.build(Path("unused"), "1.0.0")
        self.assertEqual(run.call_count, 1)
        with patch.object(distribution, "run", side_effect=["", "a" * 40, "linux\narm64\nlinux\namd64"]) as run, \
             self.assertRaisesRegex(ValueError, "native target"):
            distribution.build(Path("unused"), "1.0.0")
        self.assertEqual(run.call_count, 3)

    def test_builder_injects_version_disables_cgo_and_uses_versioned_name(self):
        distribution = module("distribution")
        with tempfile.TemporaryDirectory() as d:
            repo = Path(d) / "repo"
            repo.mkdir()
            for name in distribution.DOCS:
                path = repo / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b"docs")
            def fake_run(args, **kwargs):
                if args[:3] == ["git", "status", "--porcelain"]:
                    return ""
                if args[:3] == ["git", "rev-parse", "HEAD"]:
                    return "a" * 40
                if args[:2] == ["go", "env"]:
                    return "linux\namd64\nlinux\namd64"
                if args == ["go", "version"]:
                    return "go version go1.26.3 linux/amd64"
                if args[:2] == ["go", "build"]:
                    self.assertEqual(kwargs["env"]["CGO_ENABLED"], "0")
                    self.assertIn("-X pockettopo-exporter/internal/cli.Version=1.0.0", args)
                    Path(args[args.index("-o") + 1]).write_bytes(b"binary")
                    return ""
                self.fail("unexpected command: " + repr(args))
            output = Path(d) / "output"
            with patch.object(distribution, "REPO", repo), patch.object(distribution, "run", side_effect=fake_run), \
                 patch.object(distribution, "smoke", return_value={"native_runtime": True}):
                distribution.build(output, "1.0.0")
            archive = output / "pockettopo-exporter-1.0.0-linux-amd64.zip"
            metadata = json.loads(distribution.read_package(archive, "a" * 40, "linux-amd64", "1.0.0")["BUILD.json"])
            self.assertEqual((metadata["cli_version"], metadata["cgo_enabled"]), ("1.0.0", 0))
            report = json.loads((output / "validation.json").read_text())
            self.assertEqual(report["archive_sha256"], hashlib.sha256(archive.read_bytes()).hexdigest())
            self.assertEqual(report["archive"], archive.name)

    def test_builder_rejects_invalid_version_before_running_tools(self):
        distribution = module("distribution")
        with patch.object(distribution, "run") as run:
            for version in ("", "v1.0.0", "01.0.0", "1.0", "1.0.0/foreign", "1.0.0\n"):
                with self.subTest(version=version), self.assertRaises(ValueError):
                    distribution.build(Path("unused"), version)
            run.assert_not_called()


class EvidenceControls(unittest.TestCase):
    def test_missing_changed_and_duplicate_evidence_fails(self):
        evidence = module("verify-evidence")
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "native.txt").write_bytes(b"native\r\n")
            item = {"path": "native.txt", "sha256": hashlib.sha256(b"native\r\n").hexdigest()}
            evidence.verify([item], root)
            with self.assertRaises(ValueError):
                evidence.verify([item, item], root)
            (root / "native.txt").write_bytes(b"native\n")
            with self.assertRaises(ValueError):
                evidence.verify([item], root)
            (root / "native.txt").unlink()
            with self.assertRaises(ValueError):
                evidence.verify([item], root)


if __name__ == "__main__":
    unittest.main()
