#!/usr/bin/env python3
"""Author small P15 source cases. Expectations are checked with PocketTopo/Walls."""
import struct
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent / "internal/exportwalls/testdata"

def text(value):
    value = value.encode("utf-8")
    length, result = len(value), bytearray()
    while length >= 128:
        result.append((length & 127) | 128)
        length >>= 7
    return bytes(result) + bytes([length]) + value

def document(rows, declinations=(0, 910, 910, -32768), references=()):
    data = b"Top\x03" + struct.pack("<i", len(declinations))
    for i, decl in enumerate(declinations):
        data += struct.pack("<q", 638000000000000000 + i) + text("trip " + str(i)) + struct.pack("<h", decl)
    data += struct.pack("<i", len(rows))
    for start, end, length, azimuth, inclination, flags, trip, comment in rows:
        data += struct.pack("<IIihhBBh", start, end, length, azimuth if azimuth < 32768 else azimuth - 65536, inclination, flags, 173, trip)
        if flags & 2:
            data += text(comment)
    data += struct.pack("<i", len(references))
    for station, east, north, altitude, comment in references:
        data += struct.pack("<Iqqi", station, east, north, altitude) + text(comment)
    data += struct.pack("<iii", 0, 0, 500) * 2 + b"\0" + struct.pack("<iii", 0, 0, 500) + b"\0" + struct.pack("<i", 0)
    return data

def row(start, end, length, azimuth=0, inclination=0, flags=0, trip=0, comment=""):
    return start, end, length, azimuth, inclination, flags, trip, comment

if __name__ == "__main__":
    ROOT.mkdir(parents=True, exist_ok=True)
    cases = {
        "features": [row(65536,65537,10020,16384,flags=2,comment='zażółć\r\n#Units Decl=99'),row(65536,65537,10000,16384,flags=1),row(65537,65536,9980,49152),row(65537,0x80000000,2500),row(65537,0x80000001,5000,32768,trip=1),row(0x80000001,0,3000,8192,trip=2),row(0,65538,2000,0,16384,trip=3),row(65538,65539,0,trip=-1),row(65539,65540,1000,trip=0)],
        "mixed": [row(65536,65537,10020,0,100,trip=0),row(65536,65537,10000,1000,-100,trip=1),row(65537,65536,9980,32768,0,trip=3),row(65537,0x80000000,1000,65535),row(65537,65538,1,65535)],
        "ties": [row(65536,65537,0),row(65537,65536,0,32768),row(65537,65538,1000,0,-16384)],
        "aliases": [row(0x80200000,65536,1000),row(0x000fffff,65536,2000),row(65536,0x800ffffe,1000)],
        "mixed-zero": [row(65536,65537,0),row(65536,65537,1000)],
        "cancel": [row(65536,65537,1000),row(65536,65537,1000,32768)],
        "negative": [row(65536,65537,-1)],
        "inclination": [row(65536,65537,1000,0,16385)],
        "inclination-hidden": [row(65536,65537,1000),row(65536,65537,1000,0,16385)],
        "negative-inclination-hidden": [row(65536,65537,1000),row(65536,65537,1000,0,-16385)],
        "undefined": [row(0x80000000,65537,1000)],
        "self": [row(65536,65536,1000)],
        "self-zero": [row(65536,65536,0)],
        "alias-self-zero": [row(0x80200000,0x000fffff,0)],
        "empty": [],
    }
    for name, rows in cases.items():
        refs = [(65536,123456,654321,1000,"reference\n#Fix injected")] if name == "features" else []
        (ROOT / (name + ".top")).write_bytes(document(rows, references=refs))
