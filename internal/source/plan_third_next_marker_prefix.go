package source

import "bytes"

type PlanThirdNextMarkerPrefixOffsets struct {
	PlanThirdPolygonColorPrefixOffsets
	ThirdNextMarker Span
}

// PlanThirdNextMarkerPrefix ends after the following raw marker, including 0, before any payload.
type PlanThirdNextMarkerPrefix struct {
	color     PlanThirdPolygonColorPrefix
	marker    byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanThirdNextMarkerPrefix(color PlanThirdPolygonColorPrefix, marker byte, span Span, prefix []byte, tailBytes int) PlanThirdNextMarkerPrefix {
	return PlanThirdNextMarkerPrefix{color: color, marker: marker, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanThirdNextMarkerPrefix) ThirdNextMarkerRaw() byte { return p.marker }
func (p PlanThirdNextMarkerPrefix) ThirdColorRaw() byte      { return p.color.ThirdColorRaw() }
func (p PlanThirdNextMarkerPrefix) FollowingMarkerRaw() byte { return p.color.FollowingMarkerRaw() }
func (p PlanThirdNextMarkerPrefix) SecondColorRaw() byte     { return p.color.SecondColorRaw() }
func (p PlanThirdNextMarkerPrefix) SecondPointCountRaw() int32 {
	return p.color.SecondPointCountRaw()
}
func (p PlanThirdNextMarkerPrefix) SecondPoints() []PolygonPoint { return p.color.SecondPoints() }
func (p PlanThirdNextMarkerPrefix) ColorRaw() byte               { return p.color.ColorRaw() }
func (p PlanThirdNextMarkerPrefix) NextMarkerRaw() byte          { return p.color.NextMarkerRaw() }
func (p PlanThirdNextMarkerPrefix) ThirdPointCountRaw() int32 {
	return p.color.ThirdPointCountRaw()
}
func (p PlanThirdNextMarkerPrefix) ThirdPoints() []PolygonPoint { return p.color.ThirdPoints() }
func (p PlanThirdNextMarkerPrefix) Points() []PolygonPoint      { return p.color.Points() }
func (p PlanThirdNextMarkerPrefix) Header() [4]byte             { return p.color.Header() }
func (p PlanThirdNextMarkerPrefix) Version() byte               { return p.color.Version() }
func (p PlanThirdNextMarkerPrefix) TripCountRaw() int32         { return p.color.TripCountRaw() }
func (p PlanThirdNextMarkerPrefix) Trips() []Trip               { return p.color.Trips() }
func (p PlanThirdNextMarkerPrefix) MeasurementCountRaw() int32 {
	return p.color.MeasurementCountRaw()
}
func (p PlanThirdNextMarkerPrefix) Measurements() []Measurement { return p.color.Measurements() }
func (p PlanThirdNextMarkerPrefix) ReferenceCountRaw() int32    { return p.color.ReferenceCountRaw() }
func (p PlanThirdNextMarkerPrefix) References() []Reference     { return p.color.References() }
func (p PlanThirdNextMarkerPrefix) OverviewMapping() Mapping    { return p.color.OverviewMapping() }
func (p PlanThirdNextMarkerPrefix) PlanMapping() Mapping        { return p.color.PlanMapping() }
func (p PlanThirdNextMarkerPrefix) MarkerRaw() byte             { return p.color.MarkerRaw() }
func (p PlanThirdNextMarkerPrefix) PointCountRaw() int32        { return p.color.PointCountRaw() }
func (p PlanThirdNextMarkerPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanThirdNextMarkerPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanThirdNextMarkerPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanThirdNextMarkerPrefix) Offsets() PlanThirdNextMarkerPrefixOffsets {
	return PlanThirdNextMarkerPrefixOffsets{PlanThirdPolygonColorPrefixOffsets: p.color.Offsets(), ThirdNextMarker: p.span}
}
