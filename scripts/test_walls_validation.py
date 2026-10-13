"""Negative controls for the original Walls database verifier (no consumer oracle)."""
import importlib.util
import math
from pathlib import Path
import struct
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("walls", Path(__file__).with_name("validate-walls-ntv.py"))
walls = importlib.util.module_from_spec(spec)
spec.loader.exec_module(walls)

def ntv():
    raw = bytearray(705 + 120)
    raw[0] = 3
    struct.pack_into("<IHHH", raw, 4, 1, 705, 120, 0x0201)
    for i,(name,width) in enumerate(walls.NTV_FIELDS):
        start = 32 + 32 * i
        raw[start:start+len(name)] = name.encode()
        raw[start+11] = ord("C")
        raw[start+16] = width
    raw[704] = 13
    raw[705] = 0x20
    raw[713:721] = b"1.0     "
    raw[721:729] = b"1.1     "
    struct.pack_into("<ddd",raw,729,10,0,0)
    struct.pack_into("<I",raw,809,9)
    return raw

class WallsControls(unittest.TestCase):
    def test_requires_complete_native_schema_and_values(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d)/"native.ntv"
            good = ntv()
            path.write_bytes(good)
            rows = walls.read_ntv(path)
            self.assertEqual(rows,[{"from":"1.0","to":"1.1","vector_enu_metres":(10,0,0),"srv_line":9}])
            for index,value in ((0,0),(12,0),(14,1),(704,0),(43,ord("N")),(48,4),(705,0x40),(706,1)):
                bad = good.copy();bad[index] = value;path.write_bytes(bad)
                with self.assertRaises(ValueError):
                    walls.read_ntv(path)
            for bad in (good[:20],good[:-1],good+b"garbage"):
                path.write_bytes(bad)
                with self.assertRaises(ValueError):
                    walls.read_ntv(path)
            bad = good.copy();struct.pack_into("<d",bad,729,math.nan);path.write_bytes(bad)
            with self.assertRaises(ValueError):
                walls.read_ntv(path)

    def test_requires_final_survey_summary(self):
        with tempfile.TemporaryDirectory() as d:
            path=Path(d)/"native.ntn"
            raw=bytearray(577+96)
            raw[0]=3
            struct.pack_into("<IHH",raw,4,1,577,96)
            for i,(name,width) in enumerate(walls.NTN_FIELDS):
                start=32+32*i
                raw[start:start+len(name)]=name.encode()
                raw[start+11]=ord("C");raw[start+16]=width
            raw[576]=13;raw[577]=0x20
            raw[665:673]=b"survey  "
            struct.pack_into("<I",raw,581,3)
            struct.pack_into("<d",raw,609,12.5)
            path.write_bytes(raw)
            self.assertEqual(walls.survey_summary(path),(3,12.5))
            for index,value in ((0,0),(14,1),(43,ord("N")),(48,2),(576,0),(577,0),(665,ord("x"))):
                bad=raw.copy();bad[index]=value;path.write_bytes(bad)
                with self.assertRaises(ValueError):walls.survey_summary(path)
            path.write_bytes(raw[:-1])
            with self.assertRaises(ValueError):walls.survey_summary(path)
            for value in (math.nan,math.inf,-1):
                bad=raw.copy();struct.pack_into("<d",bad,609,value);path.write_bytes(bad)
                with self.assertRaises(ValueError):walls.survey_summary(path)

    def test_declination_and_splay_statistics(self):
        rows,splays,length=walls.expected(b";#Date YYYY-MM-DD\r\n#Units Decl=90.000000\r\n1.0 1.1 10.000 0.000000 0.000000\r\n1.1 - 2.500 90.000000 0.000000\r\n#Units Decl=0.000000\r\n1.1 1.2 1.000 0.000000 -90.000000\r\n")
        self.assertEqual((splays,length),(1,11.0))
        self.assertAlmostEqual(rows[0]["vector_enu_metres"][0],10)
        self.assertAlmostEqual(rows[1]["vector_enu_metres"][2],-1)
        self.assertEqual([x["srv_line"] for x in rows],[3,6])

if __name__ == "__main__":unittest.main()
