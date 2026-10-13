#!/usr/bin/env python3
"""P16 independent SVX acceptance by exactly Survex 1.4.22.

Analytic ENU coordinates are independent of PocketTopo geometry and the exporter.
Consumer adjustment is not a native PocketTopo compatibility claim.
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


def decode(dump):
    nodes = {}
    anonymous = []
    for line in dump.splitlines():
        match = re.fullmatch(r'NODE ([-.\d]+) ([-.\d]+) ([-.\d]+) \[([^]]*)\](.*)', line)
        if match:
            point = (tuple(map(float, match.group(1, 2, 3))), match[5])
            if not match[4]:
                anonymous.append(point)
                continue
            if match[4] in nodes:
                raise ValueError('Duplicate decoded station')
            nodes[match[4]] = point
    return nodes, anonymous


def verify_anonymous(points, expected):
    if len(points) != len(expected) or any('ANON' not in flags or 'WALL' in flags for _, flags in points):
        raise ValueError('Anonymous points lost or inferred as walls')
    remaining = list(points)
    for xyz in expected:
        for i, (point, _) in enumerate(remaining):
            if all(abs(a - b) < 0.011 for a, b in zip(point, xyz)):
                remaining.pop(i)
                break
        else:
            raise ValueError('Independent splay coordinates differ')


def validate(binary, cavern, dump3d):
    version = run([cavern, '--version'])
    if version.returncode or not re.fullmatch(rb'[^\r\n]+ - Survex 1\.4\.22', version.stdout.strip()):
        raise ValueError('SVX consumer must be exactly Survex 1.4.22')
    fixture = ROOT / 'internal/exportsurvex/testdata'
    results = []
    with tempfile.TemporaryDirectory(prefix='pockettopo-survex-') as scratch:
        scratch = Path(scratch)
        for case in ('features', 'splays', 'adjustment'):
            source = fixture / (case + '.top')
            before = source.read_bytes()
            package = scratch / case
            exported = run([binary, 'export', source, '--format', 'survex', '--output', package])
            if exported.returncode or exported.stderr or exported.stdout != b'Exported survey.svx and report.json\n':
                raise ValueError('Survex CLI failed: ' + repr(exported))
            if {p.name for p in package.iterdir()} != {'survey.svx', 'report.json'}:
                raise ValueError('Incomplete or extra SVX output')
            svx = (package / 'survey.svx').read_bytes()
            if svx != (fixture / (case + '-compact.svx')).read_bytes() or source.read_bytes() != before:
                raise ValueError('SVX golden or original source changed')
            report = json.loads((package / 'report.json').read_text())
            if (report['format'] != 'survex-svx'
                    or report['output_sha256'] != hashlib.sha256(svx).hexdigest()
                    or report['source_sha256'] != hashlib.sha256(before).hexdigest()
                    or len(report['measurements']) != 5):
                raise ValueError('SVX provenance mismatch')
            lines = svx.decode('ascii').splitlines()
            for i, row in enumerate(report['measurements']):
                if row['index'] != i or not lines[row['svx_line'] - 1].startswith(row['from']['survex_name'] + '\t' + row['to']['survex_name'] + '\t'):
                    raise ValueError('SVX row ownership mismatch')
            out3d = scratch / (case + '.3d')
            consumed = run([cavern, '--output=' + str(out3d), package / 'survey.svx'], env={**os.environ, 'LC_ALL': 'C'})
            diagnostic = (consumed.stdout + consumed.stderr).decode()
            warnings = 1 if case == 'splays' else 0
            if (consumed.returncode or diagnostic.count('warning:') != warnings or 'error:' in diagnostic
                    or (warnings and 'equated to itself' not in diagnostic)):
                raise ValueError('SVX consumer diagnostics: ' + diagnostic)
            dumped = run([dump3d, out3d])
            if dumped.returncode or dumped.stderr:
                raise ValueError('dump3d failed')
            dump = dumped.stdout.decode()
            nodes, anonymous = decode(dump)
            if case == 'splays':
                # Two independently anonymous endpoints cannot be merged by name.
                # A signed -22.5 degree declination cancels bearing +22.5.
                expected = {'0_0': (0, 0, 0), '0_1': (0, 1, 0), '0': (0, 1, 0)}
                verify_anonymous(anonymous, [(1, 1, 0), (0, 1, 2)])
                named = nodes
                line_count, splay_count = 3, 2
            else:
                # Equal forward/reverse observations average to 10m; unequal
                # 10/12m repeats adjust to 11m under equal consumer uncertainty.
                length = 11 if case == 'adjustment' else 10
                b = (100 + length * math.sin(math.pi / 8), -200 + length * math.cos(math.pi / 8), 3)
                c = (b[0] + math.sqrt(2), b[1], 3 + math.sqrt(2))
                expected = {'0_0': (100, -200, 3), '0_1': b, '0_2': c,
                            '32767_65535': c}
                named = nodes
                line_count, splay_count = 4, 1
                verify_anonymous(anonymous, [(c[0], c[1], c[2] - 1)])
            if set(named) != set(expected):
                raise ValueError('Lost or invented station: ' + repr(nodes))
            for name, xyz in expected.items():
                if any(abs(a - b) > 0.011 for a, b in zip(named[name][0], xyz)):
                    raise ValueError('Consumer coordinate mismatch: ' + name)
            if 'FIXED' not in nodes['0_0'][1] or dump.count(' SPLAY') != splay_count or dump.count('\nLINE ') != line_count:
                raise ValueError('Consumer lost reference, splay or repeated leg')
            # The consumer must actually reject a malformed observation.
            invalid = scratch / (case + '-invalid.svx')
            row = report['measurements'][0]
            original = lines[row['svx_line'] - 1]
            fields = original.split('\t')
            fields[2] = 'bad'
            invalid.write_bytes(svx.replace(original.encode(), '\t'.join(fields).encode(), 1))
            failed = run([cavern, '--output=' + str(scratch / (case + '-invalid.3d')), invalid])
            if failed.returncode == 0 or b'error:' not in failed.stdout + failed.stderr:
                raise ValueError('Consumer negative control did not reject invalid distance')
            results.append({'case': case, 'source_sha256': report['source_sha256'],
                            'svx_sha256': report['output_sha256'], 'named_stations': len([k for k in nodes if k]),
                            'anonymous_splays': splay_count, 'warnings': warnings, 'negative_control': True})
    return {'consumer': 'cavern 1.4.22 SVX reader', 'cases': results,
            'coordinate_tolerance_m': 0.011, 'reference_and_zero_equations': True,
            'independent_signed_declination_and_adjustment': True}


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
