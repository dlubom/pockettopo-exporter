#!/usr/bin/env python3
"""P16 hand-specified SVX and immutable native-readback input; never a gate."""
from pathlib import Path
root=Path(__file__).resolve().parent.parent
p=root/'internal/exportsurvex/testdata'; p.mkdir(parents=True,exist_ok=True)
# Reuse immutable, native-readback source bytes without regenerating P15 evidence.
for name in ['features.top','features.native.txt']:
    (p/name).write_bytes((root/'internal/exportwalls/testdata'/name).read_bytes())
svx=r'''; PocketTopo to Survex SVX; conversion profile 1.
; Recorded dates are unverified; no date or CRS is inferred.
*begin
*units default
*calibrate default
*case preserve
*truncate off
*flags not duplicate not surface not splay
*data normal from to tape compass clino
*infer equates on
; trip 0 ticks=637134336000000000 declination_raw=4096
; comment "recorded date"
; trip 1 ticks=0 declination_raw=-32768
; comment "Auto"
; reference 0
; comment "reference"
*fix 0_0 100 -200 3
; measurement 0
; comment "hello\r\n#Fix evil 0 0 0; \u017c"
*declination 22.5 degrees
0_0	0_1	10	0	0
; measurement 1
*declination 22.5 degrees
0_1	0_0	10	180	0
; measurement 2
*declination 0 degrees
0_1	0_2	2	90	45
; measurement 3
*declination 0 degrees
0_2	.	1	270	-90
; measurement 4
*declination 22.5 degrees
0_2	32767_65535	0	0.0054931640625	0.0054931640625
*end
'''
(p/'features.svx').write_bytes(svx.replace('\n','\r\n').encode('ascii'))

# Additional literal cases exercise consumer behavior beyond the shared native
# field golden. They contain the exact semantic cases their names claim.
import struct


def text(value):
    raw = value.encode('utf-8')
    assert len(raw) < 128
    return bytes([len(raw)]) + raw


def literal_source(trips, rows, refs):
    raw = bytearray(b'Top\x03' + struct.pack('<i', len(trips)))
    for ticks, comment, decl in trips:
        raw += struct.pack('<q', ticks) + text(comment) + struct.pack('<h', decl)
    raw += struct.pack('<i', len(rows))
    for f, t, d, azimuth, inclination, flags, roll, trip, comment in rows:
        raw += struct.pack('<IIihhBBh', f, t, d, azimuth, inclination, flags, roll, trip)
        if flags & 2:
            raw += text(comment)
    raw += struct.pack('<i', len(refs))
    for station, east, north, altitude, comment in refs:
        raw += struct.pack('<Iqqi', station, east, north, altitude) + text(comment)
    mapping = struct.pack('<iii', 0, 0, 1000)
    return raw + mapping + mapping + b'\0' + mapping + b'\0' + struct.pack('<i', 0)


adjustment = literal_source(
    [(637134336000000000, 'recorded date', 4096), (0, 'Auto', -32768)],
    [(0, 1, 10000, 0, 0, 2, 255, 0, 'hello\r\n#Fix evil 0 0 0; ż'),
     (1, 0, 12000, -32768, 0, 1, 0, 0, ''),
     (1, 2, 2000, 16384, 8192, 20, 7, 1, ''),
     (2, 0x80000000, 1000, -16384, -16384, 8, 128, -1, ''),
     (2, 0x7fffffff, 0, 1, 1, 0, 0, 0, '')],
    [(0, 100000, -200000, 3000, 'reference')])
(p/'adjustment.top').write_bytes(adjustment)
(p/'adjustment.svx').write_bytes(svx.replace('0_1\t0_0\t10\t180', '0_1\t0_0\t12\t180').replace('\n','\r\n').encode('ascii'))

splays = literal_source(
    [(0, 'signed declination', -4096)],
    [(0, 1, 1000, 4096, 0, 0, 0, 0, ''),
     (1, 0x80000000, 1000, 20480, 0, 0, 0, 0, ''),
     (1, 0x80000000, 2000, 4096, 16384, 0, 0, 0, ''),
     (1, 0x80000001, 0, 0, 0, 0, 0, 0, ''),
     (1, 0x80100002, 0, 0, 0, 0, 0, 0, '')],
    [(0, 0, 0, 0, '')])
splays_svx=r'''; PocketTopo to Survex SVX; conversion profile 1.
; Recorded dates are unverified; no date or CRS is inferred.
*begin
*units default
*calibrate default
*case preserve
*truncate off
*flags not duplicate not surface not splay
*data normal from to tape compass clino
*infer equates on
; trip 0 ticks=0 declination_raw=-4096
; comment "signed declination"
; reference 0
*fix 0_0 0 0 0
; measurement 0
*declination -22.5 degrees
0_0	0_1	1	22.5	0
; measurement 1
*declination -22.5 degrees
0_1	.	1	112.5	0
; measurement 2
*declination -22.5 degrees
0_1	.	2	22.5	90
; measurement 3
*declination -22.5 degrees
0_1	0	0	0	0
; measurement 4
*declination -22.5 degrees
0_1	0_1	0	0	0
*end
'''
(p/'splays.top').write_bytes(splays)
(p/'splays.svx').write_bytes(splays_svx.replace('\n','\r\n').encode('ascii'))
