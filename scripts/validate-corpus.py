#!/usr/bin/env python3
"""Account for every pinned TOP path; verify CLI packages and frozen regressions.

This is a project regression gate. Only separately captured native goldens are
original-program evidence. No neighboring SRV/SVX/DXF is used as an oracle.
"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

REPO = Path(__file__).resolve().parent.parent
EVIDENCE = REPO / "validation"
ARTIFACTS = {"text": "survey.txt", "therion": "survey.txt", "dxf": "survey.dxf",
             "visualtopo": "survey.tro", "toporobot": "survey.text", "dxf3d": "survey.dxf", "srv": "survey.srv", "svx": "survey.svx"}
HASH_FIELDS = {"text": "text", "therion": "text", "dxf": "dxf",
               "visualtopo": "output", "toporobot": "output", "dxf3d": "dxf", "srv": "output", "svx": "output"}


def sha256(raw):
    return hashlib.sha256(raw).hexdigest()


def verify_inputs(root, items):
    paths = [x["path"] for x in items]
    actual = {p.relative_to(root).as_posix() for p in root.rglob("*")
              if p.suffix.lower() == ".top"}
    if not paths or len(set(paths)) != len(paths) or actual != set(paths):
        raise ValueError("corpus inventory missing, extra or duplicate paths")
    for item in items:
        path = root / item["path"]
        if path.is_symlink() or not path.is_file() or root.resolve() not in path.resolve().parents:
            raise ValueError("corpus input is not a regular contained file")
        raw = path.read_bytes()
        blob = hashlib.sha1(b"blob " + str(len(raw)).encode() + b"\0" + raw).hexdigest()
        if (len(raw), sha256(raw), blob) != (item["size_bytes"], item["sha256"], item["git_blob"]):
            raise ValueError("corpus identity differs: " + item["path"])


def export_result(output, code, stderr, fmt, source_hash, source_size, inputs=None):
    diagnostic = stderr.decode("utf-8").strip()
    if code != 0:
        if code != 1 or output.exists():
            raise ValueError("failed export published output or crashed")
        if not diagnostic.startswith("Cannot export: "):
            raise ValueError("unclassified CLI failure: " + diagnostic)
        message = diagnostic.removeprefix("Cannot export: ")
        if not message.startswith(("resource_limit", "native_failure",
                                   "context requires ", "processing requires ",
                                   "export requires ", "closure_iteration_limit at ",
                                   "closure_ancestor at ", "closure_arithmetic at ",
                                   "projection_ancestor at ", "projection_arithmetic at ",
                                   "arithmetic_overflow at ")):
            raise ValueError("unexpected export failure: " + diagnostic)
        return {"status": "failed", "diagnostic": message}
    if diagnostic or not output.is_dir() or {p.name for p in output.iterdir()} != {ARTIFACTS[fmt], "report.json"}:
        raise ValueError("successful export has diagnostics or incomplete/extra files")
    raw = (output / ARTIFACTS[fmt]).read_bytes()
    report_raw = (output / "report.json").read_bytes()
    report = json.loads(report_raw)
    prefix = HASH_FIELDS[fmt]
    inventory = report.get("inputs", report.get("Inputs", []))
    current = [x for x in inventory if x["role"] == "current"]
    size = report.get("source_size_bytes")
    if size is None:
        if len(current) != 1 or current[0]["sha256"] != source_hash:
            raise ValueError("export current source inventory differs")
        size = current[0].get("size_bytes", current[0].get("Size"))
    if inputs is not None and fmt != "text":
        actual = [(x["name"], x["role"], x["sha256"], x.get("size_bytes", x.get("Size"))) for x in inventory]
        if actual != inputs:
            raise ValueError("export template/current inventory differs")
    if (report["schema_version"], report["source_sha256"], size,
        report[prefix + "_sha256"], report[prefix + "_size_bytes"]) != (
            1, source_hash, source_size, sha256(raw), len(raw)):
        raise ValueError("export report hashes or sizes differ")
    if fmt in ("dxf", "dxf3d") and not raw.endswith(b"0\r\nEOF\r\n"):
        raise ValueError("partial DXF accepted")
    return {"status": "supported", "output_sha256": sha256(raw),
            "output_size_bytes": len(raw), "report_sha256": sha256(report_raw)}


def invoke(binary, args):
    result = subprocess.run([str(binary), *map(str, args)], capture_output=True, timeout=30)
    if result.returncode not in (0, 1):
        raise ValueError("CLI crashed: " + result.stderr.decode("utf-8", errors="replace"))
    return result


def profiles():
    yield "text/isolated", "text", []
    for fmt, extra in [("therion", []), ("dxf", ["--view", "plan"]),
                       ("dxf", ["--view", "side"]), ("visualtopo", []),
                       ("toporobot", ["--export-time", "2026-10-13T12:34:56"]), ("dxf3d", [])]:
        view = ("-" + extra[1]) if fmt == "dxf" else ""
        for context in (False, True):
            for closed in (False, True):
                name = fmt + view + "/" + ("context" if context else "isolated") + ("-closed" if closed else "")
                yield name, fmt, extra + (["--template-directory", "{parent}"] if context else []) + (["--close-loops"] if closed else [])


def validate(binary, root, inventory, native):
    verify_inputs(root, inventory["inputs"])
    rows = []
    with tempfile.TemporaryDirectory(prefix="pockettopo-corpus-") as scratch:
        scratch = Path(scratch)
        for index, item in enumerate(inventory["inputs"]):
            source = root / item["path"]
            inspection = invoke(binary, ["inspect", source, "--json"])
            result = json.loads(inspection.stdout)
            if result["source"] != {"sha256": item["sha256"], "size_bytes": item["size_bytes"], "top_version": 3}:
                raise ValueError("inspection source identity differs")
            if (inspection.returncode == 0) != (result["complete"] and result["supported"]):
                raise ValueError("inspection exit/status differs")
            row = {"path": item["path"], "inspection": result, "exports": {}}
            for name, fmt, options in profiles():
                templates = []
                if "--template-directory" in options:
                    templates = [x for x in inventory["inputs"] if Path(x["path"]).parent == Path(item["path"]).parent and x["path"] != item["path"]]
                inputs = [(Path(x["path"]).name, "template", x["sha256"], x["size_bytes"]) for x in templates]
                inputs.append((source.name, "current", item["sha256"], item["size_bytes"]))
                results = []
                for repeat in range(2):
                    output = scratch / "export"
                    args = ["export", source, "--format", fmt, "--output", output]
                    args += [source.parent if value == "{parent}" else value for value in options]
                    run = invoke(binary, args)
                    current = export_result(output, run.returncode, run.stderr, fmt, item["sha256"], item["size_bytes"], inputs)
                    if current["status"] == "supported":
                        if not run.stdout.strip():
                            raise ValueError("successful export lacks notification")
                        if name == "text/isolated" and native[index]["status"] == "complete":
                            want = (EVIDENCE / "native-text" / native[index]["output"]).read_bytes()
                            if (output / ARTIFACTS[fmt]).read_bytes() != want:
                                raise ValueError("original-program corpus bytes differ: " + item["path"])
                        for p in output.iterdir():
                            p.unlink()
                        output.rmdir()
                    elif run.stdout:
                        raise ValueError("failed export wrote stdout")
                    results.append(current)
                if results[0] != results[1]:
                    raise ValueError("nondeterministic export/report: " + item["path"] + " " + name)
                row["exports"][name] = results[0]
            rows.append(row)
            if index % 25 == 0 or index == len(inventory["inputs"]) - 1:
                print("Validated", index + 1, "/", len(inventory["inputs"]), flush=True)
    verify_inputs(root, inventory["inputs"])
    return {"schema_version": 1, "corpus_commit": inventory["commit"],
            "oracle": "project regression; native-text comparisons separately identified", "inputs": rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--corpus", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--record", action="store_true", help="author a new baseline; never used in CI")
    args = parser.parse_args()
    inventory = json.loads((EVIDENCE / "corpus-inputs.json").read_text())
    native = json.loads((EVIDENCE / "native-text.json").read_text())["inputs"]
    if [x["path"] for x in native] != [x["path"] for x in inventory["inputs"]]:
        raise ValueError("native corpus inventory differs")
    report = validate(args.binary.resolve(), args.corpus.resolve(), inventory, native)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    expected = EVIDENCE / "corpus-results.json"
    if args.record:
        expected.write_bytes(args.report.read_bytes())
    elif report != json.loads(expected.read_text()):
        raise ValueError("corpus outcomes changed; inspect the report, do not regenerate the baseline blindly")
    print("Corpus gate passed:", Counter(x["inspection"]["status"] for x in report["inputs"]))


if __name__ == "__main__":
    main()
