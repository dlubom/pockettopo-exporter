package top_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func assertSideMarker(t *testing.T, b []byte, p source.SideMarkerPrefix) {
	t.Helper()
	side, err := top.ReadV3SideMappingPrefix(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.SideMappingPrefix, side) || p.Offsets().SideMappingPrefixOffsets != side.Offsets() {
		t.Fatal("lost inherited metadata")
	}
	_, mapping := side.SideMapping()
	m, present := p.SideMarkerRaw()
	end := side.ConsumedOffset()
	if mapping {
		if !present || m != b[end] || p.Offsets().SideMarker != (source.Span{Start: end, End: end + 1}) {
			t.Fatal("marker bits/presence/span")
		}
		end++
	} else if present || m != 0 || p.Offsets().SideMarker != (source.Span{}) {
		t.Fatal("marker after unsupported plan")
	}
	if p.ConsumedOffset() != end || p.UnparsedTailSize() != len(b)-end || !bytes.Equal(p.Bytes(), b[:end]) {
		t.Fatal("prefix/tail accounting")
	}
}

func TestSideMarkerPrefixEveryMarkerStopsAndCopies(t *testing.T) {
	for _, n := range []int{0, 1, 4, 17, 257} {
		for marker := 0; marker < 256; marker++ {
			for _, tail := range [][]byte{nil, {255, 3, 1, 0}} {
				b := append(sideData(n), byte(marker))
				b = append(b, tail...)
				before := bytes.Clone(b)
				p, err := top.ReadV3SideMarkerPrefix(b)
				if err != nil {
					t.Fatal(err)
				}
				assertSideMarker(t, b, p)
				if !bytes.Equal(b, before) {
					t.Fatal("input mutation")
				}
				out := p.Bytes()
				out[0] = 255
				b[0] = 255
				if !bytes.Equal(p.Bytes(), before[:len(before)-len(tail)]) {
					t.Fatal("byte alias")
				}
				elems := p.Elements()
				if len(elems) > 0 {
					elems[0] = source.PlanElement{}
					if p.Elements()[0].MarkerRaw() != 1 {
						t.Fatal("element alias")
					}
				}
			}
		}
	}
	for marker := 0; marker < 256; marker++ {
		if marker == 0 || marker == 1 || marker == 3 {
			continue
		}
		for _, tail := range [][]byte{nil, {0}, sideRecord(1, 2, 3)} {
			b := append(mixedData(4, byte(marker)), tail...)
			p, err := top.ReadV3SideMarkerPrefix(b)
			if err != nil {
				t.Fatal(err)
			}
			assertSideMarker(t, b, p)
		}
	}
}

func TestSideMarkerPrefixEveryCutAndInheritedErrors(t *testing.T) {
	limits := top.DefaultPlanElementLimits()
	for _, n := range []int{0, 4, 17} {
		b := append(sideData(n), 0)
		for cut := 0; cut < len(b); cut++ {
			before := bytes.Clone(b[:cut])
			p, err := top.ReadV3SideMarkerPrefixWithLimits(b[:cut], limits)
			if err == nil || !reflect.DeepEqual(p, source.SideMarkerPrefix{}) || !bytes.Equal(before, b[:cut]) {
				t.Fatal("partial failure")
			}
			if cut < len(b)-1 {
				_, want := top.ReadV3SideMappingPrefixWithLimits(b[:cut], limits)
				if !reflect.DeepEqual(err, want) {
					t.Fatal("inherited error")
				}
			} else {
				var pe *top.ParseError
				if !errors.As(err, &pe) || pe.Code != "truncated" || pe.Field != "side.elements[0].kind" || pe.Offset != cut {
					t.Fatalf("missing marker: %v", err)
				}
			}
		}
	}
	for _, bad := range [][]byte{{255}, fixture(-1), referenceTable(-1), append(planMarkerBase(), 1, 255, 255, 255, 255), append(planMarkerBase(), 3)} {
		_, want := top.ReadV3SideMappingPrefixWithLimits(bad, limits)
		p, err := top.ReadV3SideMarkerPrefixWithLimits(bad, limits)
		if want == nil || !reflect.DeepEqual(err, want) || !reflect.DeepEqual(p, source.SideMarkerPrefix{}) {
			t.Fatal("malformed inheritance")
		}
	}
}

func TestSideMarkerPrefixLimits(t *testing.T) {
	b := append(sideData(4), 255)
	defaults := top.DefaultPlanElementLimits()
	for i := 0; i < 8; i++ {
		for _, value := range []int{-1, 0, 1, int(^uint(0) >> 1)} {
			l := defaults
			switch i {
			case 0:
				l.MaxPoints = value
			case 1:
				l.MaxElements = value
			case 2:
				l.MaxTotalPoints = value
			case 3:
				l.MaxReferences = value
			case 4:
				l.MaxMeasurements = value
			case 5:
				l.MaxTrips = value
			case 6:
				l.MaxInputBytes = value
			case 7:
				l.MaxCommentBytes = value
			}
			_, want := top.ReadV3SideMappingPrefixWithLimits(b, l)
			p, err := top.ReadV3SideMarkerPrefixWithLimits(b, l)
			if !reflect.DeepEqual(err, want) {
				t.Fatal("limit propagation")
			}
			if err != nil {
				if !reflect.DeepEqual(p, source.SideMarkerPrefix{}) {
					t.Fatal("partial limits")
				}
			} else {
				assertSideMarker(t, b, p)
			}
		}
	}
	l := defaults
	l.MaxPoints = -1
	l.MaxElements = -1
	l.MaxTotalPoints = -1
	l.MaxInputBytes = -1
	_, want := top.ReadV3SideMappingPrefixWithLimits(b, l)
	_, err := top.ReadV3SideMarkerPrefixWithLimits(b, l)
	if !reflect.DeepEqual(err, want) {
		t.Fatal("limit precedence")
	}
	l = defaults
	l.MaxElements = 4
	l.MaxPoints = 1
	l.MaxTotalPoints = 2
	l.MaxInputBytes = len(b)
	if _, err := top.ReadV3SideMarkerPrefixWithLimits(b, l); err != nil {
		t.Fatal(err)
	}
	l.MaxInputBytes--
	if _, err := top.ReadV3SideMarkerPrefixWithLimits(b, l); err == nil {
		t.Fatal("whole input bound")
	}
	l = defaults
	l.MaxElements = 0
	l.MaxPoints = 0
	l.MaxTotalPoints = 0
	if _, err := top.ReadV3SideMarkerPrefixWithLimits(append(sideData(0), 1), l); err != nil {
		t.Fatal("side marker charged against plan budget", err)
	}
}

func TestSideMarkerPrefixVariableTables(t *testing.T) {
	b := binary.LittleEndian.AppendUint32(fixture(1, record(17, "trip", -32768)), 1)
	b = append(b, measurementRecord(255, "shot")...)
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = append(b, referenceRecord("Aą")...)
	b = append(b, planMappingRecord()...)
	b = append(b, planMappingBase()[16:]...)
	b = append(b, loopXSection(-1, 2, 0x800fffff, -2)...)
	b = append(b, 0)
	b = append(b, sideRecord(-1234, 5678, 0)...)
	b = append(b, 3, 255)
	p, err := top.ReadV3SideMarkerPrefix(b)
	if err != nil {
		t.Fatal(err)
	}
	assertSideMarker(t, b, p)
	raw := bytes.Clone(b)
	trips, shots, refs := p.Trips(), p.Measurements(), p.References()
	trips[0] = source.Trip{}
	shots[0] = source.Measurement{}
	refs[0] = source.Reference{}
	for i := range b {
		b[i] = 255
	}
	if !bytes.Equal(p.Bytes(), raw[:len(raw)-1]) || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" {
		t.Fatal("table alias")
	}
}

func TestSideMarkerPrefixNativeFixtures(t *testing.T) {
	for _, f := range []struct {
		name         string
		offset, tail int
		marker       byte
	}{{"api-references.top", 243, 4, 0}, {"api-drawings.top", 417, 262, 1}} {
		b, err := os.ReadFile("testdata/" + f.name)
		if err != nil {
			t.Fatal(err)
		}
		p, err := top.ReadV3SideMarkerPrefix(b)
		if err != nil {
			t.Fatal(err)
		}
		assertSideMarker(t, b, p)
		m, ok := p.SideMarkerRaw()
		if !ok || m != f.marker || p.Offsets().SideMarker != (source.Span{Start: f.offset, End: f.offset + 1}) || p.UnparsedTailSize() != f.tail {
			t.Fatal("fixture offset/marker/tail")
		}
	}
}

func FuzzReadV3SideMarkerPrefix(f *testing.F) {
	for _, b := range [][]byte{append(sideData(0), 0), append(sideData(4), 1), mixedData(3, 2), sideData(3), {255}} {
		f.Add(b)
	}
	l := top.DefaultPlanElementLimits()
	l.MaxInputBytes = 4096
	l.MaxTrips = 32
	l.MaxMeasurements = 32
	l.MaxReferences = 32
	l.MaxCommentBytes = 256
	l.MaxPoints = 32
	l.MaxElements = 32
	l.MaxTotalPoints = 128
	f.Fuzz(func(t *testing.T, b []byte) {
		before := bytes.Clone(b)
		p, err := top.ReadV3SideMarkerPrefixWithLimits(b, l)
		q, again := top.ReadV3SideMarkerPrefixWithLimits(b, l)
		if !bytes.Equal(before, b) || !reflect.DeepEqual(p, q) || !reflect.DeepEqual(err, again) {
			t.Fatal("mutation/determinism")
		}
		if err != nil {
			var pe *top.ParseError
			if !errors.As(err, &pe) || pe.Offset < 0 || pe.Offset > len(b) || !reflect.DeepEqual(p, source.SideMarkerPrefix{}) {
				t.Fatal("invalid error")
			}
			return
		}
		assertSideMarker(t, b, p)
	})
}
