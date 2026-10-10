package source

import "bytes"

type PlanThirdPolygonCountPrefixOffsets struct {
	PlanFollowingMarkerPrefixOffsets
	ThirdPointCount Span
}

// PlanThirdPolygonCountPrefix ends after the third Polygon's raw count.
// It contains no third points or color, including when the count is zero.
type PlanThirdPolygonCountPrefix struct {
	next      PlanFollowingMarkerPrefix
	count     int32
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanThirdPolygonCountPrefix(next PlanFollowingMarkerPrefix, count int32, span Span, prefix []byte, tailBytes int) PlanThirdPolygonCountPrefix {
	return PlanThirdPolygonCountPrefix{next: next, count: count, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanThirdPolygonCountPrefix) FollowingMarkerRaw() byte     { return p.next.FollowingMarkerRaw() }
func (p PlanThirdPolygonCountPrefix) SecondColorRaw() byte         { return p.next.SecondColorRaw() }
func (p PlanThirdPolygonCountPrefix) SecondPointCountRaw() int32   { return p.next.SecondPointCountRaw() }
func (p PlanThirdPolygonCountPrefix) SecondPoints() []PolygonPoint { return p.next.SecondPoints() }
func (p PlanThirdPolygonCountPrefix) ThirdPointCountRaw() int32    { return p.count }
func (p PlanThirdPolygonCountPrefix) NextMarkerRaw() byte          { return p.next.NextMarkerRaw() }
func (p PlanThirdPolygonCountPrefix) ColorRaw() byte               { return p.next.ColorRaw() }
func (p PlanThirdPolygonCountPrefix) Points() []PolygonPoint       { return p.next.Points() }
func (p PlanThirdPolygonCountPrefix) Header() [4]byte              { return p.next.Header() }
func (p PlanThirdPolygonCountPrefix) Version() byte                { return p.next.Version() }
func (p PlanThirdPolygonCountPrefix) TripCountRaw() int32          { return p.next.TripCountRaw() }
func (p PlanThirdPolygonCountPrefix) Trips() []Trip                { return p.next.Trips() }
func (p PlanThirdPolygonCountPrefix) MeasurementCountRaw() int32 {
	return p.next.MeasurementCountRaw()
}
func (p PlanThirdPolygonCountPrefix) Measurements() []Measurement { return p.next.Measurements() }
func (p PlanThirdPolygonCountPrefix) ReferenceCountRaw() int32    { return p.next.ReferenceCountRaw() }
func (p PlanThirdPolygonCountPrefix) References() []Reference     { return p.next.References() }
func (p PlanThirdPolygonCountPrefix) OverviewMapping() Mapping    { return p.next.OverviewMapping() }
func (p PlanThirdPolygonCountPrefix) PlanMapping() Mapping        { return p.next.PlanMapping() }
func (p PlanThirdPolygonCountPrefix) MarkerRaw() byte             { return p.next.MarkerRaw() }
func (p PlanThirdPolygonCountPrefix) PointCountRaw() int32        { return p.next.PointCountRaw() }
func (p PlanThirdPolygonCountPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanThirdPolygonCountPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanThirdPolygonCountPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanThirdPolygonCountPrefix) Offsets() PlanThirdPolygonCountPrefixOffsets {
	return PlanThirdPolygonCountPrefixOffsets{PlanFollowingMarkerPrefixOffsets: p.next.Offsets(), ThirdPointCount: p.span}
}
