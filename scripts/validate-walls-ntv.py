#!/usr/bin/env python3
"""Verify fresh original Walls NTV vectors, independently of Go geometry.

Packed layout is from original v2.3.0-beta.1 ntfile.cpp/wall_srv.h, commit
6ebe3331fd6054b830f31437d2c9e5b2e34c8881. No Survex result is accepted here.
"""
import argparse
import hashlib
import json
import math
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent
NTV_FIELDS = (("_T_DATE",3),("_W_FR_PFX",2),("_W_TO_PFX",2),("FR_NAM",8),("TO_NAM",8),
              ("_D_EAST",8),("_D_NORTH",8),("_D_UP",8),("_D_VARH",8),("_D_VARV",8),
              ("_L_FR_ID",4),("_L_TO_ID",4),("_L_STR_ID",4),("_L_STR_NXT",4),
              ("_D_A_EAST",8),("_D_A_NORTH",8),("_D_A_UP",8),("_U_LINENO",4),
              ("_W_SEG_ID",2),("_W_FILEID",2),("FILE_NAM",8))
NTN_FIELDS = (("_B_FLAG",1),("_W_NUM_SYS",2),("_U_VECTORS",4),("_L_VEC_ID",4),
              ("_L_STR_ID1",4),("_L_STR_ID2",4),("_U_PLT_ID1",4),("_U_PLT_ID2",4),
              ("_L_ENDPTS",4),("_D_LENGTH",8),("_D_E_MIN",8),("_D_N_MIN",8),
              ("_D_U_MIN",8),("_D_E_MAX",8),("_D_N_MAX",8),("_D_U_MAX",8),("NAME",8))

def schema(raw, fields):
    for i, (name, width) in enumerate(fields):
        field = raw[32 + 32 * i:64 + 32 * i]
        if (field[:11].split(b"\0",1)[0].decode("ascii"),field[11],field[16],field[17]) != (name,ord("C"),width,0):
            raise ValueError("native field schema differs")

def sha256(raw):
    return hashlib.sha256(raw).hexdigest()

def read_ntv(path):
    raw = path.read_bytes()
    if len(raw) < 32:
        raise ValueError("incomplete native NTV header")
    count, header, size = struct.unpack_from("<IHH", raw, 4)
    if (size != 120 or header != 705 or count > 1_000_000 or raw[14] != 0
            or struct.unpack_from("<H", raw, 12)[0] != 0x0201
            or raw[header - 1] != 13 or len(raw) < header + count * size):
        raise ValueError("incomplete or unsupported native NTV layout")
    end = header + count * size
    if raw[0] != 3 or raw[end:] not in (b"", b"\x1a"):
        raise ValueError("invalid native NTV file type or trailing data")
    schema(raw, NTV_FIELDS)
    records = []
    for i in range(count):
        record = raw[header + i * size:header + (i + 1) * size]
        if record[0] & 0x40:
            raise ValueError("unexpected inferred native fix")
        if record[1:4] != bytes(3):
            raise ValueError("commented date changed native measurements")
        names = [record[j:j + 8].decode("ascii").rstrip() for j in (8, 16)]
        vector = struct.unpack_from("<ddd", record, 24)
        line = struct.unpack_from("<I", record, 104)[0]
        if not all(math.isfinite(x) for x in vector):
            raise ValueError("nonfinite native vector")
        records.append({"from": names[0], "to": names[1], "vector_enu_metres": vector, "srv_line": line})
    return records

def survey_summary(path):
    raw = path.read_bytes()
    if len(raw) < 32:
        raise ValueError("incomplete native NTN header")
    count, header, size = struct.unpack_from("<IHH", raw, 4)
    if (size != 96 or header != 577 or count > 1_000_000 or raw[14] != 0
            or raw[header - 1] != 13 or len(raw) < header + count * size):
        raise ValueError("incomplete native NTN database")
    if raw[0] != 3 or raw[header + count * size:] not in (b"", b"\x1a"):
        raise ValueError("invalid native NTN file type or trailing data")
    schema(raw, NTN_FIELDS)
    surveys = []
    for i in range(count):
        record = raw[header + i * size:header + (i + 1) * size]
        if record[0] == 0x20 and record[88:96].decode("ascii").rstrip() == "survey":
            surveys.append((struct.unpack_from("<I", record, 4)[0], struct.unpack_from("<d", record, 32)[0]))
    if len(surveys) != 1:
        raise ValueError("missing final native survey summary")
    if not math.isfinite(surveys[0][1]) or surveys[0][1] < 0:
        raise ValueError("invalid native survey length")
    return surveys[0]

def expected(srv):
    decl = 0.0
    rows = []
    splays = 0
    length = 0.0
    for line, text in enumerate(srv.decode("ascii").splitlines(), 1):
        if text.startswith("#Units Decl="):
            decl = float(text.split("=", 1)[1])
        if not text or text[0] in ";#":
            continue
        start, end, dist, azimuth, inclination = text.split()
        dist, azimuth, inclination = float(dist), float(azimuth), float(inclination)
        if end == "-":
            splays += 1
            continue
        length += dist
        azimuth, inclination = math.radians(azimuth + decl), math.radians(inclination)
        rows.append({"from": start, "to": end, "srv_line": line,
                     "vector_enu_metres": (dist * math.cos(inclination) * math.sin(azimuth),
                                           dist * math.cos(inclination) * math.cos(azimuth),
                                           dist * math.sin(inclination))})
    return rows, splays, length

def validate(directory, walls):
    cases = []
    for name in ("features", "mixed", "ties", "aliases"):
        case = directory / name
        srvs = (case / "survey.srv").read_bytes()
        frozen = ROOT / "internal/exportwalls/testdata" / (name + ".srv")
        if srvs != frozen.read_bytes():
            raise ValueError("native consumer did not receive the reviewed SRV: " + name)
        source = (ROOT / "internal/exportwalls/testdata" / (name + ".top")).read_bytes()
        report = json.loads((case / "export/report.json").read_bytes())
        if (report["source_sha256"], report["output_sha256"]) != (sha256(source), sha256(srvs)):
            raise ValueError("native input/report identity differs")
        native = {}
        for extension in ("ntv", "ntn", "ntp", "nts"):
            paths = list(case.rglob("*." + extension))
            if len(paths) != 1:
                raise ValueError("missing or duplicate native workfile: " + extension)
            native[extension] = paths[0]
        actual = read_ntv(native["ntv"])
        want, splays, length = expected(srvs)
        vectors, compiled_length = survey_summary(native["ntn"])
        if vectors != len(want) or abs(compiled_length - length) > 0.00001:
            raise ValueError("incomplete native compilation or cave length includes splays")
        if len(actual) != len(want):
            raise ValueError(f"native named-vector count differs: {name}: {len(actual)} != {len(want)}")
        for got, desired in zip(actual, want):
            if (got["from"], got["to"], got["srv_line"]) != (desired["from"], desired["to"], desired["srv_line"]):
                raise ValueError("native line or station identity differs: " + name)
            if any(abs(a - b) > 0.00001 for a, b in zip(got["vector_enu_metres"], desired["vector_enu_metres"])):
                raise ValueError("native manual-declination vector differs: " + name)
        files = {p.relative_to(case).as_posix(): sha256(p.read_bytes()) for p in case.rglob("*") if p.is_file()}
        cases.append({"name": name, "source_sha256": sha256(source), "srv_sha256": sha256(srvs),
                      "named_vectors": len(actual), "cave_length_metres": compiled_length, "anonymous_wall_shots": splays,
                      "vectors": actual, "files": files})
    if list((directory / "negative-control").rglob("*.ntn")):
        raise ValueError("native negative control produced a review database")
    return {"schema_version": 1, "consumer": "original Walls v2.3.0-beta.1",
            "walls_sha256": sha256(walls.read_bytes()),
            "source_commit": "6ebe3331fd6054b830f31437d2c9e5b2e34c8881",
            "settings": "default project; no geographic reference; manual Decl; commented dates; no loops in fixtures",
            "tolerance_metres": 0.00001, "cases": cases, "negative_control": True,
            "anonymous_limit": "Walls treats '-' as anonymous wall measurements; these do not enter the named-vector NTV table or cave-length statistics. No wall point identity or LRUD semantics is inferred."}

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--walls", type=Path, required=True)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    result = validate(args.directory, args.walls)
    args.report.write_text(json.dumps(result, indent=2) + "\n")
    print("Original Walls validation passed:", len(result["cases"]), "cases and native negative control")
