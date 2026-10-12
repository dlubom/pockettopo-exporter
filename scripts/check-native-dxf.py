#!/usr/bin/env python3
"""Independent coordinate/structure checks and SVG preview of frozen native DXF."""
from collections import Counter
import argparse
import html
import json
from pathlib import Path


def entities(path):
    raw = path.read_bytes()
    assert raw.endswith(b"EOF\r\n"), path
    assert raw.count(b"\n") == raw.count(b"\r\n"), path
    assert not raw.startswith(b"\xef\xbb\xbf"), path
    lines = raw.decode("utf-8").splitlines()
    assert len(lines) % 2 == 0, path
    pairs = list(zip(lines[::2], lines[1::2]))
    assert pairs[:6] == [("0", "SECTION"), ("2", "HEADER"), ("9", "$ACADVER"), ("1", "AC1002"), ("9", "$INSUNITS"), ("70", "4")]
    start = pairs.index(("2", "ENTITIES"))
    out, current = [], None
    for code, value in pairs[start + 1:]:
        if code == "0":
            if current:
                out.append(current)
            current = {"type": value}
        elif current is not None:
            current[code] = value
    return out


def coord(e, index=0):
    return [float(e[str(10 + index)]), float(e[str(20 + index)])]


def preview(root, output):
    colors = {"0": "#111827", "1": "#dc2626", "3": "#16a34a", "5": "#2563eb", "9": "#6b7280", "30": "#ea580c", "32": "#92400e", "255": "#9ca3af"}
    parts = ['<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 540"><rect width="1000" height="540" fill="white"/><style>text{font-family:sans-serif}</style>']
    for panel, suffix in enumerate(["0", "1"]):
        es = entities(root / ("nativecontext-template-rays-" + suffix + ".dxf"))
        points = [coord(e) for e in es if "10" in e]
        points += [coord(e, 1) for e in es if "11" in e]
        minx, maxx = min(p[0] for p in points), max(p[0] for p in points)
        miny, maxy = min(p[1] for p in points), max(p[1] for p in points)
        scale = min(420 / (maxx - minx), 410 / (maxy - miny))
        def xy(point):
            return (panel * 500 + 40 + (point[0] - minx) * scale, 480 - (point[1] - miny) * scale)
        parts.append(f'<text x="{panel * 500 + 30}" y="30" font-size="20">Native DXF — {"plan" if panel == 0 else "side"}</text>')
        poly, polycolor = [], "#111827"
        for e in es:
            kind, color = e["type"], colors.get(e.get("62", "0"), "#111827")
            if kind == "POLYLINE":
                poly, polycolor = [], color
            elif kind == "VERTEX":
                poly.append(xy(coord(e)))
            elif kind == "SEQEND" and poly:
                pts = " ".join(f"{x:.3f},{y:.3f}" for x, y in poly)
                parts.append(f'<polyline points="{pts}" fill="none" stroke="{polycolor}" stroke-width="2"/>')
            elif kind == "LINE":
                x, y = xy(coord(e)); x2, y2 = xy(coord(e, 1))
                dash = ' stroke-dasharray="5,4"' if e.get("6") == "DASH" else ""
                parts.append(f'<line x1="{x:.3f}" y1="{y:.3f}" x2="{x2:.3f}" y2="{y2:.3f}" stroke="{color}" stroke-width="2"{dash}/>')
            elif kind in ("POINT", "CIRCLE"):
                x, y = xy(coord(e)); radius = max(2, float(e.get("40", "0")) * scale)
                parts.append(f'<circle cx="{x:.3f}" cy="{y:.3f}" r="{radius:.3f}" fill="none" stroke="{color}" stroke-width="1.5"/>')
            elif kind == "TEXT":
                x, y = xy(coord(e))
                parts.append(f'<text x="{x+5:.3f}" y="{y-5:.3f}" font-size="12">{html.escape(e["1"])}</text>')
        parts.append(f'<text x="{panel*500+30}" y="515" font-size="11">Sketch; current survey in red; template in orange; dashed connectors.</text><text x="{panel*500+30}" y="533" font-size="11">Coordinates read directly from frozen native DXF. Y is north/up.</text>')
    parts.append("</svg>\n")
    output.write_text("".join(parts))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path("internal/exportdxf/testdata"))
    parser.add_argument("--report", type=Path, default=Path("build/p10/native-coordinate-checks.json"))
    parser.add_argument("--preview", type=Path, default=Path("build/p10/native-dxf-preview.svg"))
    args = parser.parse_args(); root = args.root
    counts = Counter()
    for path in root.glob("*.dxf"):
        if path.name in ("zero-polygons-0.dxf", "zero-polygons-1.dxf", "missing-reference-geo.dxf") or path.name.startswith("nativecontext-boundary-data-geo-"):
            assert not path.read_bytes().endswith(b"EOF\r\n")
            counts["partial_outputs"] += 1
        else:
            counts["complete_outputs"] += 1
            entities(path)
    plain = entities(root / "features-o0.dxf")
    geo = entities(root / "features-o512.dxf")
    assert coord(plain[0]) == [2.01, 6.01]
    assert coord(geo[0]) == [202.02, 406.02]
    assert Counter(e["type"] for e in plain)["CIRCLE"] == 0
    assert Counter(e["type"] for e in plain)["POINT"] == 128
    assert Counter(e["type"] for e in plain)["POLYLINE"] == 128
    allopt = entities(root / "features-o3582.dxf")
    colors = {e.get("62") for e in allopt if e["type"] in ("POLYLINE", "POINT")}
    assert colors == {"0", "1", "3", "5", "9", "30", "32"}
    assert {e.get("8") for e in allopt} >= {"Shots", "XSect", "Grid", "Label", "LGray", "Green", "Black", "Red", "Brown", "Orange", "Gray", "Blue"}
    grid = [e for e in allopt if e.get("8") == "Grid" and e["type"] == "LINE"]
    feetgrid = [e for e in entities(root / "features-feet-0.dxf") if e.get("8") == "Grid" and e["type"] == "LINE"]
    assert round(coord(grid[1])[0] - coord(grid[0])[0], 6) == 2
    assert round(coord(feetgrid[1])[0] - coord(feetgrid[0])[0], 6) == 3.04
    connectors = [e for e in entities(root / "nativecontext-template-rays-0.dxf") if e["type"] == "LINE" and e.get("8") == "LGray"]
    assert coord(connectors[0]) == [22.552, 13.316]
    assert coord(connectors[0], 1) == [20.542, 11.306]
    assert connectors[0]["6"] == "DASH"
    assert coord(entities(root / "features-scale3-0.dxf")[0]) == [335, 1001.66666666667]
    negative_grid = [e for e in entities(root / "grid-negative-metres-o1024.dxf") if e["type"] == "LINE"]
    negative_feet = [e for e in entities(root / "grid-negative-feet-o1024.dxf") if e["type"] == "LINE"]
    actual_grid = [e for e in entities(root / "nativecontext-grid-template-bounds-metres-o1024.dxf") if e["type"] == "LINE"]
    template_grid = [e for e in entities(root / "nativecontext-grid-template-bounds-metres-o1040.dxf") if e["type"] == "LINE"]
    assert len(negative_grid) == 13
    assert coord(negative_grid[0]) == [-6.0, 8.0]
    assert coord(negative_grid[0], 1) == [-6.0, -4.0]
    assert coord(negative_feet[0]) == [-6.08, 9.12]
    assert len(actual_grid) == 10 and len(template_grid) == 16
    assert coord(actual_grid[0]) == [-4.0, 14.0]
    assert coord(actual_grid[0], 1) == [-4.0, 6.0]
    assert coord(template_grid[0]) == [-6.0, 14.0]
    assert coord(template_grid[0], 1) == [-6.0, -4.0]
    report = dict(counts)
    report.update({"first_sketch_point": coord(plain[0]), "first_geographic_point": coord(geo[0]), "metre_grid_step": 2, "foot_grid_step": 3.04, "template_connector": [coord(connectors[0]), coord(connectors[0], 1)], "negative_bound_grid_start": coord(negative_grid[0]), "negative_bound_feet_grid_start": coord(negative_feet[0]), "current_only_grid_lines": len(actual_grid), "current_and_template_grid_lines": len(template_grid), "all_raw_colors": 256, "sketch_single_points": 128, "sketch_polylines": 128})
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    preview(root, args.preview)
    print(json.dumps(report))


if __name__ == "__main__":
    main()
