package top

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
)

// Independent layout: empty tables, overview, empty plan, side mapping.
func documentBytes(side, tail []byte) []byte {
	b := append([]byte{'T', 'o', 'p', 3}, make([]byte, 36)...)
	b = append(b, 0)
	b = append(b, make([]byte, 12)...)
	b = append(b, side...)
	return append(b, tail...)
}
func docPolygon(n int, color byte) []byte {
	b := []byte{1}
	b = binary.LittleEndian.AppendUint32(b, uint32(n))
	for i := 0; i < n; i++ {
		b = binary.LittleEndian.AppendUint32(b, 0xfffefdfd+uint32(i))
		b = binary.LittleEndian.AppendUint32(b, 0x01000001+uint32(i))
	}
	return append(b, color)
}
func docXSection() []byte {
	b := []byte{3}
	for _, v := range []uint32{0x80000000, 0x7fffffff, 0x800fffff, 0xfffffffe} {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return b
}
func TestDocumentSupported(t *testing.T) {
	for color := 0; color < 256; color++ {
		side := append(docPolygon(2, byte(color)), docXSection()...)
		side = append(side, docPolygon(0, byte(color))...)
		side = append(side, 0)
		b := documentBytes(side, []byte{0, 0, 0, 0})
		binary.LittleEndian.PutUint32(b[41:45], 0xfffefdfd)
		binary.LittleEndian.PutUint32(b[45:49], 0x01000001)
		binary.LittleEndian.PutUint32(b[49:53], 0xfffffe0b)
		original := bytes.Clone(b)
		d, err := ReadV3(b)
		if err != nil {
			t.Fatal(err)
		}
		if !d.Complete() || !d.Supported() || d.TailKind() != source.TailZeroTrailer || d.ConsumedOffset() != len(b) || d.UnparsedTailSize() != 0 || d.BodyEnd() != len(b)-4 {
			t.Fatal("accounting", d)
		}
		if d.Version() != 3 || d.Header() != [4]byte{'T', 'o', 'p', 3} || len(d.Trips()) != 0 || len(d.Measurements()) != 0 || len(d.References()) != 0 || d.TableOffsets().ReferenceCount != (source.Span{Start: 12, End: 16}) || d.Overview().Offsets().Record != (source.Span{Start: 16, End: 28}) {
			t.Fatal("tables")
		}
		plan, present := d.Plan()
		if !present || plan.Mapping().Offsets().Record != (source.Span{Start: 28, End: 40}) || plan.StopSpan() != (source.Span{Start: 40, End: 41}) || plan.StopMarkerRaw() != 0 || plan.TotalPoints() != 0 || plan.StopReason() != source.PlanStopTerminated {
			t.Fatal("plan")
		}
		drawing, present := d.Side()
		if !present || drawing.Mapping().X0Raw() != -66051 || drawing.Mapping().Y0Raw() != 16777217 || drawing.Mapping().ScaleRaw() != -501 || drawing.TotalPoints() != 2 || drawing.Mapping().Offsets() != (source.MappingOffsets{Record: source.Span{Start: 41, End: 53}, X0: source.Span{Start: 41, End: 45}, Y0: source.Span{Start: 45, End: 49}, Scale: source.Span{Start: 49, End: 53}}) {
			t.Fatal("side")
		}
		es := drawing.Elements()
		if len(es) != 3 || es[0].MarkerRaw() != 1 || es[1].MarkerRaw() != 3 {
			t.Fatal("order")
		}
		p, ok := es[0].Polygon()
		if !ok || p.ColorRaw() != byte(color) || p.PointCountRaw() != 2 || p.Offsets() != (source.PlanPolygonOffsets{Record: source.Span{Start: 53, End: 75}, Marker: source.Span{Start: 53, End: 54}, PointCount: source.Span{Start: 54, End: 58}, Points: source.Span{Start: 58, End: 74}, Color: source.Span{Start: 74, End: 75}}) {
			t.Fatal("polygon")
		}
		pts := p.Points()
		if pts[0].XRaw() != -66051 || pts[0].YRaw() != 16777217 || pts[1].Offsets() != (source.PolygonPointOffsets{Record: source.Span{Start: 66, End: 74}, X: source.Span{Start: 66, End: 70}, Y: source.Span{Start: 70, End: 74}}) {
			t.Fatal("points", pts)
		}
		x, ok := es[1].XSection()
		if !ok || x.CenterXRaw() != -2147483648 || x.CenterYRaw() != 2147483647 || x.StationID().Raw() != 0x800fffff || x.DirectionRaw() != -2 || x.Offsets() != (source.PlanXSectionOffsets{Record: source.Span{Start: 75, End: 92}, Marker: source.Span{Start: 75, End: 76}, CenterX: source.Span{Start: 76, End: 80}, CenterY: source.Span{Start: 80, End: 84}, StationID: source.Span{Start: 84, End: 88}, Direction: source.Span{Start: 88, End: 92}}) {
			t.Fatal("xs", x)
		}
		es[0] = source.PlanElement{}
		pts[0] = source.PolygonPoint{}
		b[0] = 0
		raw := d.Bytes()
		raw[1] = 0
		tail := d.UnparsedBytes()
		if len(tail) != 0 || !bytes.Equal(d.Bytes(), original) {
			t.Fatal("input alias")
		}
		again, _ := d.Side()
		q, _ := again.Elements()[0].Polygon()
		if q.Points()[0].XRaw() != -66051 {
			t.Fatal("accessor alias")
		}
	}
}
func TestDocumentTail(t *testing.T) {
	for _, tc := range []struct {
		tail      []byte
		kind      source.TailKind
		supported bool
		consumed  int
	}{
		{nil, source.TailAbsent, true, 54}, {[]byte{0, 0, 0, 0}, source.TailZeroTrailer, true, 58},
		{[]byte{0}, source.TailPartial, false, 54}, {[]byte{0, 0}, source.TailPartial, false, 54}, {[]byte{0, 0, 0}, source.TailPartial, false, 54},
		{[]byte{1, 0, 0, 0}, source.TailNonzero, false, 54}, {[]byte{0, 0, 0, 1}, source.TailNonzero, false, 54},
		{[]byte{0, 0, 0, 0, 255}, source.TailExtra, false, 58}, {[]byte{1, 0, 0, 0, 255}, source.TailNonzero, false, 54},
	} {
		b := documentBytes([]byte{0}, tc.tail)
		d, e := ReadV3(b)
		if e != nil || !d.Complete() || d.Supported() != tc.supported || d.TailKind() != tc.kind || d.ConsumedOffset() != tc.consumed || d.ConsumedOffset()+d.UnparsedTailSize() != len(b) || !bytes.Equal(d.UnparsedBytes(), b[tc.consumed:]) {
			t.Fatalf("tail %v: %+v %v", tc.tail, d, e)
		}
		u := d.UnparsedBytes()
		if len(u) > 0 {
			u[0] ^= 255
			if bytes.Equal(u, d.UnparsedBytes()) {
				t.Fatal("tail alias")
			}
		}
	}
}
func TestDocumentUnknown(t *testing.T) {
	for marker := 1; marker < 256; marker++ {
		if marker == 1 || marker == 3 {
			continue
		}
		b := documentBytes([]byte{byte(marker), 9, 8, 7}, nil)
		d, e := ReadV3(b)
		s, ok := d.Side()
		if e != nil || !ok || d.Complete() || d.Supported() || d.TailKind() != source.TailUnreached || s.StopMarkerRaw() != byte(marker) || s.StopReason() != source.PlanStopUnsupportedElement || d.ConsumedOffset() != 54 || d.UnparsedTailSize() != 3 {
			t.Fatal(marker, d, e)
		}
	}
	b := documentBytes([]byte{0}, nil)
	b[40] = 255
	d, e := ReadV3(b)
	_, ok := d.Side()
	if e != nil || ok || d.Complete() || d.Supported() || d.BodyEnd() != 41 || d.ConsumedOffset() != 41 || d.UnparsedTailSize() != 13 {
		t.Fatal(d, e)
	}
}
func TestDocumentErrors(t *testing.T) {
	b := documentBytes(append(append(docPolygon(2, 255), docXSection()...), 0), nil)
	for cut := 0; cut < len(b); cut++ {
		d, e := ReadV3(b[:cut])
		var pe *ParseError
		if !errors.As(e, &pe) || pe.Code != "truncated" || !reflect.DeepEqual(d, source.Document{}) {
			t.Fatalf("cut %d: %v %v", cut, d, e)
		}
	}
	for _, tc := range []struct {
		side        []byte
		code, field string
		offset      int
	}{
		{[]byte{1, 255, 255, 255, 255}, "negative_count", "side.elements[0].point_count", 54},
		{[]byte{1, 1, 0, 0, 0}, "truncated", "side.elements[0].points", 58},
		{docPolygon(2, 255)[:13], "truncated", "side.elements[0].points", 58},
		{[]byte{1, 0, 0, 0, 0}, "truncated", "side.elements[0].color", 58},
	} {
		d, e := ReadV3(documentBytes(tc.side, nil))
		var pe *ParseError
		if !errors.As(e, &pe) || *pe != (ParseError{tc.code, tc.field, tc.offset}) || !reflect.DeepEqual(d, source.Document{}) {
			t.Fatal(d, e)
		}
	}
}
func TestDocumentLimits(t *testing.T) {
	empty := documentBytes([]byte{0}, nil)
	l := DefaultDocumentLimits()
	l.MaxElements = 0
	l.MaxTotalPoints = 0
	l.MaxDocumentElements = 0
	l.MaxDocumentPoints = 0
	l.MaxPoints = 0
	if _, e := ReadV3WithLimits(empty, l); e != nil {
		t.Fatal(e)
	}
	b := documentBytes(append(append(docPolygon(1, 255), docXSection()...), 0), nil)
	for _, tc := range []struct {
		name        string
		change      func(*DocumentLimits)
		code, field string
		offset      int
	}{
		{"element zero", func(l *DocumentLimits) { l.MaxElements = 0 }, "resource_limit", "side.elements[0].kind", 53},
		{"element one", func(l *DocumentLimits) { l.MaxElements = 1 }, "resource_limit", "side.elements[1].kind", 67},
		{"document element", func(l *DocumentLimits) { l.MaxDocumentElements = 1 }, "resource_limit", "side.elements[1].kind", 67},
		{"points", func(l *DocumentLimits) { l.MaxPoints = 0 }, "resource_limit", "side.elements[0].point_count", 54},
		{"drawing points", func(l *DocumentLimits) { l.MaxTotalPoints = 0 }, "resource_limit", "side.elements[0].point_count", 54},
		{"document points", func(l *DocumentLimits) { l.MaxDocumentPoints = 0 }, "resource_limit", "side.elements[0].point_count", 54},
		{"input", func(l *DocumentLimits) { l.MaxInputBytes = len(b) - 1 }, "resource_limit", "input", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := DefaultDocumentLimits()
			tc.change(&l)
			_, e := ReadV3WithLimits(b, l)
			var pe *ParseError
			if !errors.As(e, &pe) || *pe != (ParseError{tc.code, tc.field, tc.offset}) {
				t.Fatal(e)
			}
		})
	}
	for _, tc := range []struct {
		max int
		set func(*DocumentLimits, int)
	}{
		{100000, func(l *DocumentLimits, n int) { l.MaxElements = n }},
		{1000000, func(l *DocumentLimits, n int) { l.MaxTotalPoints = n }},
		{1000000, func(l *DocumentLimits, n int) { l.MaxPoints = n }},
		{100000, func(l *DocumentLimits, n int) { l.MaxDocumentElements = n }},
		{1000000, func(l *DocumentLimits, n int) { l.MaxDocumentPoints = n }},
	} {
		for _, n := range []int{-1, tc.max + 1} {
			limits := DefaultDocumentLimits()
			tc.set(&limits, n)
			_, e := ReadV3WithLimits(empty, limits)
			var pe *ParseError
			if !errors.As(e, &pe) || pe.Code != "invalid_limit" {
				t.Fatal(e)
			}
		}
	}
	l = DefaultDocumentLimits()
	l.MaxElements = 2
	l.MaxTotalPoints = 1
	l.MaxPoints = 1
	l.MaxDocumentElements = 2
	l.MaxDocumentPoints = 1
	if _, e := ReadV3WithLimits(b, l); e != nil {
		t.Fatal(e)
	}
	// Same records in plan and side: per-drawing exact ceilings pass, document sum fails.
	both := append(bytes.Clone(b[:40]), b[53:]...)
	both = append(both, b[41:]...)
	l.MaxDocumentElements = 4
	l.MaxDocumentPoints = 2
	if _, e := ReadV3WithLimits(both, l); e != nil {
		t.Fatal(e)
	}
	l.MaxDocumentElements = 3
	if _, e := ReadV3WithLimits(both, l); e == nil {
		t.Fatal("global elements")
	}
	l.MaxDocumentElements = 4
	l.MaxDocumentPoints = 1
	if _, e := ReadV3WithLimits(both, l); e == nil {
		t.Fatal("global points")
	}
}
func TestDocumentNativeFixtures(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		body, sideCount, points int
	}{{"api-references.top", 244, 0, 0}, {"api-drawings.top", 676, 10, 22}} {
		b, e := os.ReadFile("testdata/" + tc.name)
		if e != nil {
			t.Fatal(e)
		}
		d, e := ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		s, ok := d.Side()
		if !ok || !d.Supported() || d.BodyEnd() != tc.body || len(s.Elements()) != tc.sideCount || s.TotalPoints() != tc.points || d.UnparsedTailSize() != 0 {
			t.Fatalf("%s: body %d records %d points %d", tc.name, d.BodyEnd(), len(s.Elements()), s.TotalPoints())
		}
	}
}
func FuzzDocument(f *testing.F) {
	f.Add(documentBytes(append(docPolygon(2, 255), 0), nil))
	f.Add(documentBytes([]byte{255}, []byte{8}))
	f.Add([]byte("Top\x03"))
	f.Fuzz(func(t *testing.T, b []byte) {
		d, e := ReadV3(b)
		if e == nil {
			if d.ConsumedOffset()+d.UnparsedTailSize() != len(b) || !bytes.Equal(d.Bytes(), b) {
				t.Fatal("accounting")
			}
			if d.Supported() && !d.Complete() {
				t.Fatal("status")
			}
		}
	})
}
func BenchmarkDocument(b *testing.B) {
	data := documentBytes(append(docPolygon(100000, 255), 0), []byte{0, 0, 0, 0})
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e := ReadV3(data); e != nil {
			b.Fatal(e)
		}
	}
}

func TestDocumentLoopBudgets(t *testing.T) {
	side := append(append(append(docPolygon(1, 7), docXSection()...), docPolygon(2, 128)...), docPolygon(1, 255)...)
	side = append(side, 0)
	b := documentBytes(side, nil)
	l := DefaultDocumentLimits()
	l.MaxElements = 4
	l.MaxTotalPoints = 4
	l.MaxPoints = 2
	l.MaxDocumentElements = 4
	l.MaxDocumentPoints = 4
	d, e := ReadV3WithLimits(b, l)
	s, _ := d.Side()
	if e != nil || s.TotalPoints() != 4 || len(s.Elements()) != 4 {
		t.Fatal(d, e)
	}
	for _, mutate := range []func(*DocumentLimits){func(l *DocumentLimits) { l.MaxTotalPoints = 3 }, func(l *DocumentLimits) { l.MaxDocumentPoints = 3 }, func(l *DocumentLimits) { l.MaxElements = 3 }, func(l *DocumentLimits) { l.MaxDocumentElements = 3 }} {
		limits := l
		mutate(&limits)
		_, e := ReadV3WithLimits(b, limits)
		var pe *ParseError
		if !errors.As(e, &pe) || pe.Code != "resource_limit" {
			t.Fatal("aggregate bypass", e)
		}
	}
	b[len(b)-1] = 255
	d, e = ReadV3WithLimits(b, l)
	if e != nil || d.ConsumedOffset() != len(b) || d.UnparsedTailSize() != 0 || d.Complete() {
		t.Fatal("unknown after exact budget", e)
	}
	for _, mutate := range []func(*DocumentLimits){func(l *DocumentLimits) { l.MaxTrips = -1 }, func(l *DocumentLimits) { l.MaxMeasurements = -1 }, func(l *DocumentLimits) { l.MaxReferences = -1 }, func(l *DocumentLimits) { l.MaxCommentBytes = -1 }} {
		limits := l
		mutate(&limits)
		_, e := ReadV3WithLimits(b, limits)
		var pe *ParseError
		if !errors.As(e, &pe) || pe.Code != "invalid_limit" {
			t.Fatal("inherited validation", e)
		}
	}
}

func BenchmarkDocumentElements(b *testing.B) {
	var side []byte
	for i := 0; i < 100000; i++ {
		side = append(side, 1, 0, 0, 0, 0, 255)
	}
	side = append(side, 0)
	data := documentBytes(side, nil)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e := ReadV3(data); e != nil {
			b.Fatal(e)
		}
	}
}
