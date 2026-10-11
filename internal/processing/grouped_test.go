package processing

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) source.Document {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestGroupedNativeGoldens(t *testing.T) {
	for _, name := range []string{"empty", "repeats", "boundaries", "identities", "flags", "random"} {
		t.Run(name, func(t *testing.T) {
			d := readFixture(t, name)
			before := d.Bytes()
			r, e := GroupMeasurements(d)
			if e != nil {
				t.Fatal(e)
			}
			rows, groups := r.Rows(), r.Groups()
			b, e := os.ReadFile("testdata/" + name + ".tsv")
			if e != nil {
				t.Fatal(e)
			}
			if len(rows) != len(d.Measurements()) {
				t.Fatalf("rows=%d, want %d", len(rows), len(d.Measurements()))
			}
			gi, ri := 0, 0
			for _, line := range strings.FieldsFunc(string(b), func(r rune) bool { return r == '\r' || r == '\n' }) {
				if strings.HasPrefix(line, "row\t") {
					var i int
					var az int16
					var v Vector
					if n, e := fmt.Sscanf(line, "row\t%d\t%d\t%d\t%d\t%d\t%d", &i, &az, &v.DX, &v.DY, &v.DH, &v.DD); e != nil || n != 6 {
						t.Fatal(line, e)
					}
					row := rows[i]
					if row.Index != i || row.RealAzimuth != az || row.Vector != v || row.Source != d.Measurements()[i] {
						t.Fatalf("row %d: %+v native=%s", i, row, line)
					}
					ri++
				} else {
					var start, end int
					var kind string
					var v Vector
					var bits [6]uint32
					if n, e := fmt.Sscanf(line, "group\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%X\t%X\t%X\t%X\t%X\t%X", &start, &end, &kind, &v.DX, &v.DY, &v.DH, &v.DD, &bits[0], &bits[1], &bits[2], &bits[3], &bits[4], &bits[5]); e != nil || n != 13 {
						t.Fatal(line, e)
					}
					if gi >= len(groups) {
						t.Fatalf("missing native group %s", line)
					}
					g := groups[gi]
					gi++
					if g.HeadIndex != start || g.Kind != kind || g.Vector != v || len(g.Members()) != end-start {
						t.Fatalf("group: %+v native=%s", g, line)
					}
					head := rows[start].Source
					if g.From != head.From() || g.To != head.To() || g.Declination != rows[start].Declination {
						t.Fatalf("head lineage: %+v", g)
					}
					for j, m := range g.Members() {
						idx := start + j
						row := rows[idx]
						wantReverse := !row.Source.From().SameIdentity(head.From())
						if m.RowIndex != idx || m.Reversed != wantReverse {
							t.Fatalf("member=%+v row=%d reversed=%v", m, idx, wantReverse)
						}
						span := row.Source.Offsets().Record
						if span.Start < 0 || span.End > len(before) || span.End-span.Start < 20 {
							t.Fatal(span)
						}
					}
					got := [6]float32{g.Forward.DistanceMM, g.Forward.AzimuthRadians, g.Forward.InclinationRadians, g.Reverse.DistanceMM, g.Reverse.AzimuthRadians, g.Reverse.InclinationRadians}
					for i, val := range got {
						if math.Float32bits(val) != bits[i] {
							t.Fatalf("group %d direction[%d]=%08X native=%08X", start, i, math.Float32bits(val), bits[i])
						}
					}
				}
			}
			if ri != len(rows) || gi != len(groups) {
				t.Fatal("unaccounted rows/groups", ri, gi)
			}
			if !reflect.DeepEqual(before, d.Bytes()) {
				t.Fatal("input modified")
			}
			if !reflect.DeepEqual(r, mustGroup(t, d)) {
				t.Fatal("nondeterministic")
			}
		})
	}
}
func mustGroup(t *testing.T, d source.Document) Result {
	t.Helper()
	r, e := GroupMeasurements(d)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestGroupedCopiesAndRejectedDocument(t *testing.T) {
	d := readFixture(t, "repeats")
	r := mustGroup(t, d)
	want := mustGroup(t, d)
	rows := r.Rows()
	rows[0] = Row{}
	groups := r.Groups()
	members := groups[0].Members()
	members[0] = Member{RowIndex: -9}
	groups[0] = Group{}
	if !reflect.DeepEqual(r, want) {
		t.Fatal("accessor leaked writable state")
	}
	for _, d := range []source.Document{{}, readUnsupported(t)} {
		r, e := GroupMeasurements(d)
		if e == nil || len(r.Rows()) != 0 || len(r.Groups()) != 0 {
			t.Fatal("unsupported processed", r, e)
		}
	}
}
func readUnsupported(t *testing.T) source.Document {
	t.Helper()
	d := readFixture(t, "empty")
	b := append(d.Bytes(), 1)
	out, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return out
}

// A distinct reserved From with an unchanged To must split the pair, even
// though both From IDs have empty native display text (ID equality uses value).
func TestGroupedDistinctReservedFrom(t *testing.T) {
	d := readFixture(t, "identities")
	raw := d.Bytes()
	rows := d.Measurements()
	binary.LittleEndian.PutUint32(raw[rows[5].Offsets().To.Start:], rows[4].To().Raw())
	d, e := top.ReadV3(raw)
	if e != nil {
		t.Fatal(e)
	}
	r := mustGroup(t, d)
	found4, found5 := false, false
	for _, g := range r.Groups() {
		if g.HeadIndex == 4 {
			found4 = true
			if len(g.Members()) != 1 {
				t.Fatal("collapsed reserved From", g)
			}
		}
		if g.HeadIndex == 5 {
			found5 = true
		}
	}
	if !found4 || !found5 {
		t.Fatal("reserved pair was merged", r.Groups())
	}
}
