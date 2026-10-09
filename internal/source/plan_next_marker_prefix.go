package source

import "bytes"

type PlanNextMarkerPrefixOffsets struct {
	PlanPolygonColorPrefixOffsets
	NextMarker Span
}

// PlanNextMarkerPrefix ends after the next raw marker, including 0, before its payload.
type PlanNextMarkerPrefix struct {
	color     PlanPolygonColorPrefix
	marker    byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanNextMarkerPrefix(color PlanPolygonColorPrefix, marker byte, span Span, prefix []byte, tailBytes int) PlanNextMarkerPrefix {
	return PlanNextMarkerPrefix{color: color, marker: marker, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanNextMarkerPrefix) NextMarkerRaw() byte         { return p.marker }
func (p PlanNextMarkerPrefix) ColorRaw() byte              { return p.color.ColorRaw() }
func (p PlanNextMarkerPrefix) Points() []PolygonPoint      { return p.color.Points() }
func (p PlanNextMarkerPrefix) Header() [4]byte             { return p.color.Header() }
func (p PlanNextMarkerPrefix) Version() byte               { return p.color.Version() }
func (p PlanNextMarkerPrefix) TripCountRaw() int32         { return p.color.TripCountRaw() }
func (p PlanNextMarkerPrefix) Trips() []Trip               { return p.color.Trips() }
func (p PlanNextMarkerPrefix) MeasurementCountRaw() int32  { return p.color.MeasurementCountRaw() }
func (p PlanNextMarkerPrefix) Measurements() []Measurement { return p.color.Measurements() }
func (p PlanNextMarkerPrefix) ReferenceCountRaw() int32    { return p.color.ReferenceCountRaw() }
func (p PlanNextMarkerPrefix) References() []Reference     { return p.color.References() }
func (p PlanNextMarkerPrefix) OverviewMapping() Mapping    { return p.color.OverviewMapping() }
func (p PlanNextMarkerPrefix) PlanMapping() Mapping        { return p.color.PlanMapping() }
func (p PlanNextMarkerPrefix) MarkerRaw() byte             { return p.color.MarkerRaw() }
func (p PlanNextMarkerPrefix) PointCountRaw() int32        { return p.color.PointCountRaw() }
func (p PlanNextMarkerPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanNextMarkerPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanNextMarkerPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanNextMarkerPrefix) Offsets() PlanNextMarkerPrefixOffsets {
	return PlanNextMarkerPrefixOffsets{PlanPolygonColorPrefixOffsets: p.color.Offsets(), NextMarker: p.span}
}
