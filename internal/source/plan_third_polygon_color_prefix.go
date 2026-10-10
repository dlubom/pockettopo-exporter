package source

import "bytes"

type PlanThirdPolygonColorPrefixOffsets struct {
	PlanThirdPolygonPointsPrefixOffsets
	ThirdColor Span
}

// PlanThirdPolygonColorPrefix ends after third color, before any following marker.
type PlanThirdPolygonColorPrefix struct {
	points    PlanThirdPolygonPointsPrefix
	color     byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanThirdPolygonColorPrefix(points PlanThirdPolygonPointsPrefix, color byte, span Span, prefix []byte, tailBytes int) PlanThirdPolygonColorPrefix {
	return PlanThirdPolygonColorPrefix{points: points, color: color, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanThirdPolygonColorPrefix) ThirdColorRaw() byte      { return p.color }
func (p PlanThirdPolygonColorPrefix) FollowingMarkerRaw() byte { return p.points.FollowingMarkerRaw() }
func (p PlanThirdPolygonColorPrefix) SecondColorRaw() byte     { return p.points.SecondColorRaw() }
func (p PlanThirdPolygonColorPrefix) SecondPointCountRaw() int32 {
	return p.points.SecondPointCountRaw()
}
func (p PlanThirdPolygonColorPrefix) SecondPoints() []PolygonPoint { return p.points.SecondPoints() }
func (p PlanThirdPolygonColorPrefix) ColorRaw() byte               { return p.points.ColorRaw() }
func (p PlanThirdPolygonColorPrefix) NextMarkerRaw() byte          { return p.points.NextMarkerRaw() }
func (p PlanThirdPolygonColorPrefix) ThirdPointCountRaw() int32 {
	return p.points.ThirdPointCountRaw()
}
func (p PlanThirdPolygonColorPrefix) ThirdPoints() []PolygonPoint { return p.points.ThirdPoints() }
func (p PlanThirdPolygonColorPrefix) Points() []PolygonPoint      { return p.points.Points() }
func (p PlanThirdPolygonColorPrefix) Header() [4]byte             { return p.points.Header() }
func (p PlanThirdPolygonColorPrefix) Version() byte               { return p.points.Version() }
func (p PlanThirdPolygonColorPrefix) TripCountRaw() int32         { return p.points.TripCountRaw() }
func (p PlanThirdPolygonColorPrefix) Trips() []Trip               { return p.points.Trips() }
func (p PlanThirdPolygonColorPrefix) MeasurementCountRaw() int32 {
	return p.points.MeasurementCountRaw()
}
func (p PlanThirdPolygonColorPrefix) Measurements() []Measurement { return p.points.Measurements() }
func (p PlanThirdPolygonColorPrefix) ReferenceCountRaw() int32    { return p.points.ReferenceCountRaw() }
func (p PlanThirdPolygonColorPrefix) References() []Reference     { return p.points.References() }
func (p PlanThirdPolygonColorPrefix) OverviewMapping() Mapping    { return p.points.OverviewMapping() }
func (p PlanThirdPolygonColorPrefix) PlanMapping() Mapping        { return p.points.PlanMapping() }
func (p PlanThirdPolygonColorPrefix) MarkerRaw() byte             { return p.points.MarkerRaw() }
func (p PlanThirdPolygonColorPrefix) PointCountRaw() int32        { return p.points.PointCountRaw() }
func (p PlanThirdPolygonColorPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanThirdPolygonColorPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanThirdPolygonColorPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanThirdPolygonColorPrefix) Offsets() PlanThirdPolygonColorPrefixOffsets {
	return PlanThirdPolygonColorPrefixOffsets{PlanThirdPolygonPointsPrefixOffsets: p.points.Offsets(), ThirdColor: p.span}
}
