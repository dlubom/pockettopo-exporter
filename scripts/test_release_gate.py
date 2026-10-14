"""Reject hosted CI evidence that does not authorize the exact release revision."""
import importlib.util
from pathlib import Path
import tempfile
import unittest


def gate():
    spec = importlib.util.spec_from_file_location("release_gate", Path(__file__).with_name("release_gate.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ReleaseGateTests(unittest.TestCase):
    def test_published_release_must_be_stable_and_latest(self):
        module = gate()
        release = {"id": 17, "draft": False, "prerelease": False,
                   "tag_name": "v1.0.0", "published_at": "2026-10-14T12:00:00Z"}
        module.validate_published(release, dict(release), "v1.0.0")
        for field, value in [("draft", True), ("prerelease", True),
                             ("tag_name", "v1.0.1"), ("published_at", None)]:
            with self.subTest(field=field), self.assertRaises(ValueError):
                module.validate_published(dict(release, **{field: value}), release, "v1.0.0")
        for latest in [dict(release, tag_name="v0.9.0"), dict(release, id=18)]:
            with self.assertRaises(ValueError):
                module.validate_published(release, latest, "v1.0.0")

    def test_downloaded_assets_must_have_exact_members_and_bytes(self):
        module = gate()
        with tempfile.TemporaryDirectory() as scratch:
            expected, downloaded = Path(scratch) / "expected", Path(scratch) / "downloaded"
            expected.mkdir()
            downloaded.mkdir()
            for directory in (expected, downloaded):
                (directory / "package.zip").write_bytes(b"verified archive")
                (directory / "SHA256SUMS").write_bytes(b"checksums")
            module.validate_assets(expected, downloaded)
            (downloaded / "package.zip").write_bytes(b"foreign archive")
            with self.assertRaises(ValueError):
                module.validate_assets(expected, downloaded)
            (downloaded / "package.zip").write_bytes(b"verified archive")
            (downloaded / "extra").write_bytes(b"extra")
            with self.assertRaises(ValueError):
                module.validate_assets(expected, downloaded)
            (downloaded / "extra").unlink()
            (downloaded / "package.zip").unlink()
            with self.assertRaises(ValueError):
                module.validate_assets(expected, downloaded)

    def test_only_complete_exact_revision_main_ci_authorizes_release(self):
        module = gate()
        revision = "a" * 40
        run = {"name": "CI", "head_sha": revision, "head_branch": "main",
               "event": "push", "status": "completed", "conclusion": "success"}
        jobs = [{"name": name, "status": "completed", "conclusion": "success"}
                for name in ["CI complete", "corpus", "release-package (linux-amd64)"]]
        expected = {job["name"] for job in jobs}
        module.validate_run(run, jobs, revision, expected)
        for field, value in [("name", "Other"), ("head_sha", "b" * 40),
                             ("head_branch", "feature"), ("event", "pull_request"),
                             ("status", "in_progress"), ("conclusion", "failure")]:
            with self.subTest(field=field), self.assertRaises(ValueError):
                module.validate_run(dict(run, **{field: value}), jobs, revision, expected)
        for outcome in ("failure", "skipped", "cancelled", "neutral", None):
            with self.subTest(outcome=outcome), self.assertRaises(ValueError):
                module.validate_run(run, [dict(jobs[0], conclusion=outcome), *jobs[1:]], revision, expected)
        for damaged in (jobs[:-1], [*jobs, jobs[0]], [*jobs, {"name": "new gate", "status": "in_progress", "conclusion": None}]):
            with self.assertRaises(ValueError):
                module.validate_run(run, damaged, revision, expected)

    def test_tag_must_be_annotated_and_point_to_the_verified_commit(self):
        module = gate()
        ref = {"ref": "refs/tags/v1.0.0", "object": {"type": "tag", "sha": "b" * 40}}
        tag = {"tag": "v1.0.0", "object": {"type": "commit", "sha": "a" * 40}}
        module.validate_tag(ref, tag, "v1.0.0", "a" * 40)
        for bad_ref, bad_tag in [(dict(ref, object={"type": "commit", "sha": "a" * 40}), tag),
                                 (ref, dict(tag, object={"type": "commit", "sha": "c" * 40})),
                                 (ref, dict(tag, tag="v1.0.1"))]:
            with self.assertRaises(ValueError):
                module.validate_tag(bad_ref, bad_tag, "v1.0.0", "a" * 40)


if __name__ == "__main__":
    unittest.main()
