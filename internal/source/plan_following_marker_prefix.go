package source

import "bytes"

type PlanFollowingMarkerPrefixOffsets struct {
	PlanSecondPolygonColorPrefixOffsets
	FollowingMarker Span
}

// PlanFollowingMarkerPrefix ends after the following raw marker, including 0, before any payload.
type PlanFollowingMarkerPrefix struct {
	color     PlanSecondPolygonColorPrefix
	marker    byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanFollowingMarkerPrefix(color PlanSecondPolygonColorPrefix, marker byte, span Span, prefix []byte, tailBytes int) PlanFollowingMarkerPrefix {
	return PlanFollowingMarkerPrefix{color: color, marker: marker, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanFollowingMarkerPrefix) FollowingMarkerRaw() byte { return p.marker }
func (p PlanFollowingMarkerPrefix) SecondColorRaw() byte     { return p.color.SecondColorRaw() }
func (p PlanFollowingMarkerPrefix) ColorRaw() byte           { return p.color.ColorRaw() }
func (p PlanFollowingMarkerPrefix) NextMarkerRaw() byte      { return p.color.NextMarkerRaw() }
func (p PlanFollowingMarkerPrefix) SecondPointCountRaw() int32 {
	return p.color.SecondPointCountRaw()
}
func (p PlanFollowingMarkerPrefix) SecondPoints() []PolygonPoint { return p.color.SecondPoints() }
func (p PlanFollowingMarkerPrefix) Points() []PolygonPoint       { return p.color.Points() }
func (p PlanFollowingMarkerPrefix) Header() [4]byte              { return p.color.Header() }
func (p PlanFollowingMarkerPrefix) Version() byte                { return p.color.Version() }
func (p PlanFollowingMarkerPrefix) TripCountRaw() int32          { return p.color.TripCountRaw() }
func (p PlanFollowingMarkerPrefix) Trips() []Trip                { return p.color.Trips() }
func (p PlanFollowingMarkerPrefix) MeasurementCountRaw() int32 {
	return p.color.MeasurementCountRaw()
}
func (p PlanFollowingMarkerPrefix) Measurements() []Measurement { return p.color.Measurements() }
func (p PlanFollowingMarkerPrefix) ReferenceCountRaw() int32    { return p.color.ReferenceCountRaw() }
func (p PlanFollowingMarkerPrefix) References() []Reference     { return p.color.References() }
func (p PlanFollowingMarkerPrefix) OverviewMapping() Mapping    { return p.color.OverviewMapping() }
func (p PlanFollowingMarkerPrefix) PlanMapping() Mapping        { return p.color.PlanMapping() }
func (p PlanFollowingMarkerPrefix) MarkerRaw() byte             { return p.color.MarkerRaw() }
func (p PlanFollowingMarkerPrefix) PointCountRaw() int32        { return p.color.PointCountRaw() }
func (p PlanFollowingMarkerPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanFollowingMarkerPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanFollowingMarkerPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanFollowingMarkerPrefix) Offsets() PlanFollowingMarkerPrefixOffsets {
	return PlanFollowingMarkerPrefixOffsets{PlanSecondPolygonColorPrefixOffsets: p.color.Offsets(), FollowingMarker: p.span}
}
