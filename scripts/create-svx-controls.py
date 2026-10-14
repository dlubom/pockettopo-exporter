#!/usr/bin/env python3
"""Independently authored P16 Survex controls from the documented source fields.

These are reviewable literal polar expectations, not output from the Go writer.
Run deliberately to author controls; consumer gates never call this script.
"""
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent / 'internal/exportsvx/testdata'
HEADER = '''*require 1.4.22
*begin
*case preserve
*truncate off
*set separator :
*set names ._-
*units default
*calibrate default
*sd default
*flags not duplicate not splay not surface
*infer equates off
*infer plumbs off
*infer exports off
*data normal from to tape compass clino
*declination 0 degrees
'''
CASES = {
 'features': '''1.0 1.1 10.020 90 0
1.0 1.1 10.000 90 0
1.0 1.1 9.980 90 0
1.1 :: 2.500 0 0
*declination 4.998779 degrees
1.1 0 5.000 180 0
0 0.0 3.000 45 0
*declination 0 degrees
0.0 1.2 2.000 0 90
1.2 1.3 0.000 0 0
1.3 1.4 1.000 0 0
''',
 'mixed': '''1.0 1.1 10.020 0 0.549316
*declination 4.998779 degrees
1.0 1.1 10.000 5.493164 -0.549316
*declination 0 degrees
1.0 1.1 9.980 0 0
1.1 :: 1.000 359.994507 0
1.1 1.2 0.001 359.994507 0
''',
 'ties': '''1.0 1.1 0.000 0 0
1.0 1.1 0.000 0 0
1.1 1.2 1.000 0 -90
''',
 'aliases': '''15.65535 1.0 1.000 0 0
15.65535 1.0 2.000 0 0
1.0 p1z141z1 1.000 0 0
''',
 'interrupted': '''1.0 1.1 10.000 90 0
1.1 :: 1.000 0 0
1.1 1.0 10.000 270 0
1.0 1.2 2.000 0 0
1.0 1.1 10.000 90 0
''',
 'dates': '''*date 2020-01-01
1.0 1.1 10.020 90 0
*date 2022-01-01
1.0 1.1 10.000 90 0
*date 2024-01-01
1.0 1.1 9.980 90 0
''',
 'vertical-reverse': '''1.0 1.1 2.000 359.994507 90
1.0 1.1 2.000 359.994507 90
1.1 1.2 1.000 0 -90
1.2 :: 0.000 0 0
''',
 'reverse-head': '''1.1 1.0 10.020 270 -45
1.1 1.0 9.980 270 -45
''',
}
if __name__ == '__main__':
    for name, data in CASES.items():
        (ROOT / (name+'.control.svx')).write_text(HEADER+data+'*end\n', encoding='utf-8')
    data = CASES['dates']
    header = HEADER.replace('*declination 0 degrees\n', '*cs EPSG:32634\n*cs out EPSG:32634\n*fix 1.0 400000 5450000 1000\n*declination auto 400000 5450000 1000\n')
    (ROOT/'auto.control.svx').write_text(header+data+'*end\n',encoding='utf-8')
