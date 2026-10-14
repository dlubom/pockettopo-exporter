#!/usr/bin/env python3
"""Author P16 semantic inputs; no consumer expectations are generated here."""
import importlib.util
from pathlib import Path
ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location('srv', ROOT / 'scripts/create-srv-fixtures.py')
srv = importlib.util.module_from_spec(spec)
spec.loader.exec_module(srv)
r = srv.row
cases = {
 'interrupted': [r(65536,65537,10000,16384), r(65537,0x80000000,1000),
                 r(65537,65536,10000,49152), r(65536,65538,2000),
                 r(65536,65537,10000,16384)],
 'dates': [r(65536,65537,10020,16384,trip=0), r(65537,65536,10000,49152,trip=1),
           r(65536,65537,9980,16384,trip=2)],
 'vertical-reverse': [r(65536,65537,2000,65535,16384), r(65537,65536,2000,32767,-16384),
                      r(65537,65538,1000,0,-16384), r(65538,0x80000000,0)],
 'reverse-head': [r(65537,65536,10020,49152,-8192), r(65536,65537,9980,16384,8192)],
}
if __name__ == '__main__':
    out = ROOT / 'internal/exportsvx/testdata'
    out.mkdir(parents=True,exist_ok=True)
    for name,rows in cases.items():
        (out / (name+'.top')).write_bytes(srv.document(rows, declinations=(0,0,0)))
