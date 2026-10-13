"""Negative controls for P14 evidence and distribution integrity."""
import hashlib
import importlib.util
import json
import stat
from pathlib import Path
import tempfile
import unittest
import zipfile
import warnings


def module(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + ".py"))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


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
            archive = Path(d) / "package.zip"
            payload = {name: b"docs" for name in distribution.DOCS}
            payload["pockettopo-exporter"] = b"binary"
            metadata = {"schema_version": 1, "source_revision": "a" * 40,
                        "target": "linux-amd64", "files": {k: hashlib.sha256(v).hexdigest() for k, v in payload.items()}}
            payload["BUILD.json"] = json.dumps(metadata).encode()
            payload["SHA256SUMS"] = "".join(hashlib.sha256(v).hexdigest() + "  " + k + "\n" for k, v in sorted(payload.items())).encode()
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
