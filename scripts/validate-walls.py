#!/usr/bin/env python3
"""P15 independent SRV consumer check: exact Survex 1.4.22 Walls reader.

This validates syntax and decoded numerical geometry, not Walls GUI behavior or
native PocketTopo geometry/averaging. No SVX exporter is involved.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def run(args, **kwargs):
    return subprocess.run(list(map(str, args)), capture_output=True, timeout=60, **kwargs)


def validate(binary, cavern, dump3d):
    version = run([cavern, '--version'])
    # An uninstalled binary prints argv[0], including the supplied path.
    if version.returncode or not re.fullmatch(rb'[^\r\n]+ - Survex 1\.4\.22', version.stdout.strip()):
        raise ValueError('Walls consumer must be exactly Survex 1.4.22')
    source = ROOT / 'internal/exportwalls/testdata/features.top'
    golden = ROOT / 'internal/exportwalls/testdata/features-averaged.srv'
    before = source.read_bytes()
    with tempfile.TemporaryDirectory(prefix='pockettopo-walls-') as scratch:
        scratch = Path(scratch)
        package = scratch / 'export'
        exported = run([binary, 'export', source, '--format', 'walls', '--output', package])
        if exported.returncode or exported.stderr or exported.stdout != b'Exported survey.srv and report.json\n':
            raise ValueError('Walls CLI failed: ' + repr(exported))
        if {p.name for p in package.iterdir()} != {'survey.srv', 'report.json'}:
            raise ValueError('Incomplete or extra Walls output')
        srv = (package / 'survey.srv').read_bytes()
        if srv != golden.read_bytes() or source.read_bytes() != before:
            raise ValueError('Walls golden or original source changed')
        report = json.loads((package / 'report.json').read_text())
        if (report['output_sha256'] != hashlib.sha256(srv).hexdigest()
                or report['source_sha256'] != hashlib.sha256(before).hexdigest()
                or len(report['measurements']) != 5):
            raise ValueError('Walls provenance mismatch')
        averages = report['averages']
        if (len(averages) != 1 or averages[0]['head_index'] != 0
                or averages[0]['members'] != [{'row_index': 0, 'reversed': False},
                                              {'row_index': 1, 'reversed': True}]
                or any(report['measurements'][i]['srv_line'] != averages[0]['srv_line'] for i in (0, 1))):
            raise ValueError('Averaged leg lost raw member/line provenance')
        out3d = scratch / 'survey.3d'
        consumed = run([cavern, '--output=' + str(out3d), package / 'survey.srv'], env={**os.environ, 'LC_ALL': 'C'})
        # The reader documents GRID as unsupported; zero is its standalone default.
        # Retained plumb azimuth has one additional expected warning.
        diagnostic = (consumed.stdout + consumed.stderr).decode()
        if consumed.returncode or diagnostic.count('warning:') != 2 or 'Compass reading given on plumbed leg' not in diagnostic or 'Unknown command' not in diagnostic or 'Grid' not in diagnostic or 'error:' in diagnostic:
            raise ValueError('Consumer failed or unexpected diagnostics: ' + diagnostic)
        dumped = run([dump3d, out3d])
        if dumped.returncode:
            raise ValueError('dump3d failed')
        dump = dumped.stdout.decode()
        nodes = {}
        for line in dump.splitlines():
            match = re.fullmatch(r'NODE ([-.\d]+) ([-.\d]+) ([-.\d]+) \[([^]]*)\](.*)', line)
            if match:
                if match[4] in nodes:
                    raise ValueError('Duplicate decoded station')
                nodes[match[4]] = (tuple(map(float, match.group(1, 2, 3))), match[5])
        # Independent analytic ENU: 10m, bearing +22.5 degrees; 2m at +45
        # inclination and 90 degrees; vertical 1m splay; zero-length identity.
        b = (100 + 10 * math.sin(math.pi / 8), -200 + 10 * math.cos(math.pi / 8), 3)
        c = (b[0] + math.sqrt(2), b[1], 3 + math.sqrt(2))
        expected = {'0.0': (100, -200, 3), '0.1': b, '0.2': c,
                    'pzik0zj': c, '': (c[0], c[1], c[2] - 1)}
        if set(nodes) != set(expected):
            raise ValueError('Lost or invented station: ' + repr(nodes))
        for name, xyz in expected.items():
            if any(abs(a - b) > 0.011 for a, b in zip(nodes[name][0], xyz)):
                raise ValueError('Consumer coordinate mismatch: ' + name)
        if 'FIXED' not in nodes['0.0'][1] or 'ANON WALL' not in nodes[''][1] or dump.count(' SPLAY') != 1 or dump.count('\nLINE ') != 3:
            raise ValueError('Consumer lost reference, splay or repeated leg')
        # Independent consumer must actually reject a malformed measurement.
        invalid = scratch / 'invalid.srv'
        invalid.write_bytes(srv.replace(b'0.0\t0.1\t10\t0\t0', b'0.0\t0.1\tbad\t0\t0'))
        failed = run([cavern, '--output=' + str(scratch / 'invalid.3d'), invalid])
        if failed.returncode == 0 or b'error:' not in failed.stdout + failed.stderr:
            raise ValueError('Consumer negative control did not reject invalid distance')
        return {'consumer': 'cavern 1.4.22 Walls reader', 'source_sha256': report['source_sha256'],
                'srv_sha256': report['output_sha256'], 'named_stations': 4, 'anonymous_splays': 1,
                'coordinate_tolerance_m': 0.011, 'reference_and_zero_link': True,
                'expected_warnings': ['Grid=0 ignored by Survex (standalone zero default)', 'Compass reading given on plumbed leg'], 'negative_control': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--cavern', default='cavern')
    parser.add_argument('--dump3d', default='dump3d')
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    result = validate(args.binary.resolve(), args.cavern, args.dump3d)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result))
