package source

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDocumentModel(t *testing.T) {
	trip := NewTrip(42, "raw", -32768, TripOffsets{Record: Span{8, 22}})
	measurement := NewMeasurement(NewStationID(0xffffffff), NewStationID(0x800fffff), -1, -2, -3, 255, 128, -1, "comment", MeasurementOffsets{Record: Span{26, 55}})
	reference := NewReference(NewStationID(1), -7, 8, -9, "r", ReferenceOffsets{Record: Span{59, 85}})
	base := NewReferencePrefix(NewMeasurementPrefix(NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, []Trip{trip}, nil, 0), 1, []Measurement{measurement}, nil, 22, 0), 1, []Reference{reference}, nil, 55, 0)
	overview := NewMapping(-1, 2, -3, MappingOffsets{Record: Span{85, 97}})
	elements := []PlanElement{NewPlanXSectionElement(NewPlanXSection(3, -1, 2, NewStationID(0x800fffff), -2, PlanXSectionOffsets{Record: Span{109, 126}}))}
	drawing := NewDrawing(overview, elements, 255, Span{126, 127}, PlanStopUnsupportedElement, 42)
	raw := []byte{1, 2, 3, 4, 5}
	d := NewDocument(raw, base, overview, drawing, drawing, true, 4, 4, TailUnreached)
	raw[0] = 9
	elements[0] = PlanElement{}
	d.Trips()[0] = Trip{}
	d.Measurements()[0] = Measurement{}
	d.References()[0] = Reference{}
	d.Bytes()[0] = 8
	d.UnparsedBytes()[0] = 7
	drawing.Elements()[0] = PlanElement{}
	if !bytes.Equal(d.Bytes(), []byte{1, 2, 3, 4, 5}) || d.Header() != base.Header() || d.Version() != 3 || !reflect.DeepEqual(d.TableOffsets(), base.Offsets()) || d.Overview() != overview || d.Trips()[0] != trip || d.Measurements()[0] != measurement || d.References()[0] != reference || d.BodyEnd() != 4 || d.ConsumedOffset() != 4 || d.UnparsedTailSize() != 1 || d.TailKind() != TailUnreached || d.Complete() || d.Supported() {
		t.Fatal("document fields or alias")
	}
	p, ok := d.Plan()
	s, present := d.Side()
	if !ok || !present || p.Mapping() != overview || p.StopMarkerRaw() != 255 || p.StopSpan() != (Span{126, 127}) || p.StopReason() != PlanStopUnsupportedElement || p.TotalPoints() != 42 || s.Elements()[0].MarkerRaw() != 3 {
		t.Fatal("drawing fields or alias")
	}
	inconsistent := NewDocument([]byte{0}, base, overview, drawing, drawing, true, 1, 1, TailAbsent)
	if inconsistent.Supported() {
		t.Fatal("unsupported drawing must never be supported")
	}
	var empty Document
	if _, ok := empty.Plan(); ok {
		t.Fatal("zero plan")
	}
	if _, ok := empty.Side(); ok {
		t.Fatal("zero side")
	}
	if empty.Complete() || empty.Supported() || empty.Bytes() != nil || empty.UnparsedBytes() != nil {
		t.Fatal("zero doc")
	}
}

func TestDocumentCompletion(t *testing.T) {
	terminated := NewDrawing(Mapping{}, nil, 0, Span{0, 1}, PlanStopTerminated, 0)
	for _, tail := range []TailKind{TailAbsent, TailZeroTrailer, TailPartial, TailNonzero, TailExtra, TailUnreached} {
		d := NewDocument([]byte{0}, ReferencePrefix{}, Mapping{}, terminated, terminated, true, 1, 1, tail)
		want := tail == TailAbsent || tail == TailZeroTrailer
		if !d.Complete() || d.Supported() != want {
			t.Fatal("completion/support", tail)
		}
	}
	absent := NewDocument([]byte{0}, ReferencePrefix{}, Mapping{}, terminated, terminated, false, 1, 1, TailAbsent)
	if absent.Complete() || absent.Supported() {
		t.Fatal("missing side")
	}
}
