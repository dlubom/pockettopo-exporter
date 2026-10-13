#!/usr/bin/env python3
"""Independent literal TOP and hand-calculated SRV P15 fixtures; never a gate."""
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parent.parent / 'internal/exportwalls/testdata'


def string(value):
    data = value.encode('utf-8')
    size = len(data)
    prefix = bytearray()
    while size >= 128:
        prefix.append((size & 127) | 128)
        size >>= 7
    return bytes(prefix) + bytes([size]) + data


def fixture():
    out = bytearray(b'Top\x03')
    out += struct.pack('<i', 2)
    for ticks, comment, decl in [(637134336000000000, 'recorded date', 4096), (0, 'Auto', -32768)]:
        out += struct.pack('<q', ticks) + string(comment) + struct.pack('<h', decl)
    rows = [
        (0, 1, 10000, 0, 0, 2, 255, 0, 'hello\r\n#Fix evil 0 0 0; ż'),
        (1, 0, 10000, -32768, 0, 1, 0, 0, ''),
        (1, 2, 2000, 16384, 8192, 20, 7, 1, ''),
        (2, 0x80000000, 1000, -16384, -16384, 8, 128, -1, ''),
        (2, 0x7fffffff, 0, 1, 1, 0, 0, 0, ''),
    ]
    out += struct.pack('<i', len(rows))
    for f, t, d, a, i, flags, roll, trip, comment in rows:
        out += struct.pack('<IIihhBBh', f, t, d, a, i, flags, roll, trip)
        if flags & 2:
            out += string(comment)
    out += struct.pack('<iIqqi', 1, 0, 100000, -200000, 3000) + string('reference')
    mapping = struct.pack('<iii', 0, 0, 1000)
    out += mapping + mapping + b'\0' + mapping + b'\0' + struct.pack('<i', 0)
    return out


# Expected conversion text is intentionally literal, calculated from the fields
# above (one turn = 65536, mm / 1000). It never imports/runs the Go exporter.
SRV = r'''; PocketTopo to Walls SRV; conversion profile 1.
; Recorded dates are unverified; no date or CRS is inferred.
#Units Reset Meters Order=DAV A=D V=D Case=Mixed Grid=0
; trip 0 ticks=637134336000000000 declination_raw=4096
; comment "recorded date"
; trip 1 ticks=0 declination_raw=-32768
; comment "Auto"
; reference 0
; comment "reference"
#Fix 0.0 100 -200 3
; measurement 0
; comment "hello\r\n#Fix evil 0 0 0; \u017c"
#Units Decl=22.5
0.0	0.1	10	0	0
; measurement 1
#Units Decl=22.5
0.1	0.0	10	180	0
; measurement 2
#Units Decl=0
0.1	0.2	2	90	45
; measurement 3
#Units Decl=0
0.2	-	1	270	-90
; measurement 4
#Units Decl=22.5
0.2	pzik0zj	0	0.0054931640625	0.0054931640625
'''

if __name__ == '__main__':
    ROOT.mkdir(parents=True, exist_ok=True)
    (ROOT / 'features.top').write_bytes(fixture())
    (ROOT / 'features.srv').write_bytes(SRV.replace('\n', '\r\n').encode('ascii'))
