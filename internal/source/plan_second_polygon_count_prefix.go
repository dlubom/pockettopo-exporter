package source

import "bytes"

type PlanSecondPolygonCountPrefixOffsets struct {
	PlanNextMarkerPrefixOffsets
	SecondPointCount Span
}

// PlanSecondPolygonCountPrefix ends after the second Polygon's raw count.
// It contains no second points or color, including when the count is zero.
type PlanSecondPolygonCountPrefix struct {
	next      PlanNextMarkerPrefix
	count     int32
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanSecondPolygonCountPrefix(next PlanNextMarkerPrefix, count int32, span Span, prefix []byte, tailBytes int) PlanSecondPolygonCountPrefix {
	return PlanSecondPolygonCountPrefix{next: next, count: count, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanSecondPolygonCountPrefix) SecondPointCountRaw() int32 { return p.count }
func (p PlanSecondPolygonCountPrefix) NextMarkerRaw() byte        { return p.next.NextMarkerRaw() }
func (p PlanSecondPolygonCountPrefix) ColorRaw() byte             { return p.next.ColorRaw() }
func (p PlanSecondPolygonCountPrefix) Points() []PolygonPoint     { return p.next.Points() }
func (p PlanSecondPolygonCountPrefix) Header() [4]byte            { return p.next.Header() }
func (p PlanSecondPolygonCountPrefix) Version() byte              { return p.next.Version() }
func (p PlanSecondPolygonCountPrefix) TripCountRaw() int32        { return p.next.TripCountRaw() }
func (p PlanSecondPolygonCountPrefix) Trips() []Trip              { return p.next.Trips() }
func (p PlanSecondPolygonCountPrefix) MeasurementCountRaw() int32 {
	return p.next.MeasurementCountRaw()
}
func (p PlanSecondPolygonCountPrefix) Measurements() []Measurement { return p.next.Measurements() }
func (p PlanSecondPolygonCountPrefix) ReferenceCountRaw() int32    { return p.next.ReferenceCountRaw() }
func (p PlanSecondPolygonCountPrefix) References() []Reference     { return p.next.References() }
func (p PlanSecondPolygonCountPrefix) OverviewMapping() Mapping    { return p.next.OverviewMapping() }
func (p PlanSecondPolygonCountPrefix) PlanMapping() Mapping        { return p.next.PlanMapping() }
func (p PlanSecondPolygonCountPrefix) MarkerRaw() byte             { return p.next.MarkerRaw() }
func (p PlanSecondPolygonCountPrefix) PointCountRaw() int32        { return p.next.PointCountRaw() }
func (p PlanSecondPolygonCountPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanSecondPolygonCountPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanSecondPolygonCountPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanSecondPolygonCountPrefix) Offsets() PlanSecondPolygonCountPrefixOffsets {
	return PlanSecondPolygonCountPrefixOffsets{PlanNextMarkerPrefixOffsets: p.next.Offsets(), SecondPointCount: p.span}
}
