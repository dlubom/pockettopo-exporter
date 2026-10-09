package source

import "bytes"

type PlanPolygonCountPrefixOffsets struct {
	PlanMarkerPrefixOffsets
	PointCount Span
}

// PlanPolygonCountPrefix ends after the first Polygon's raw point count.
// It contains no points or color, including when the count is zero.
type PlanPolygonCountPrefix struct {
	marker    PlanMarkerPrefix
	count     int32
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanPolygonCountPrefix(marker PlanMarkerPrefix, count int32, span Span, prefix []byte, tailBytes int) PlanPolygonCountPrefix {
	return PlanPolygonCountPrefix{marker: marker, count: count, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}

func (p PlanPolygonCountPrefix) Header() [4]byte             { return p.marker.Header() }
func (p PlanPolygonCountPrefix) Version() byte               { return p.marker.Version() }
func (p PlanPolygonCountPrefix) TripCountRaw() int32         { return p.marker.TripCountRaw() }
func (p PlanPolygonCountPrefix) Trips() []Trip               { return p.marker.Trips() }
func (p PlanPolygonCountPrefix) MeasurementCountRaw() int32  { return p.marker.MeasurementCountRaw() }
func (p PlanPolygonCountPrefix) Measurements() []Measurement { return p.marker.Measurements() }
func (p PlanPolygonCountPrefix) ReferenceCountRaw() int32    { return p.marker.ReferenceCountRaw() }
func (p PlanPolygonCountPrefix) References() []Reference     { return p.marker.References() }
func (p PlanPolygonCountPrefix) OverviewMapping() Mapping    { return p.marker.OverviewMapping() }
func (p PlanPolygonCountPrefix) PlanMapping() Mapping        { return p.marker.PlanMapping() }
func (p PlanPolygonCountPrefix) MarkerRaw() byte             { return p.marker.MarkerRaw() }
func (p PlanPolygonCountPrefix) PointCountRaw() int32        { return p.count }
func (p PlanPolygonCountPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanPolygonCountPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanPolygonCountPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanPolygonCountPrefix) Offsets() PlanPolygonCountPrefixOffsets {
	return PlanPolygonCountPrefixOffsets{PlanMarkerPrefixOffsets: p.marker.Offsets(), PointCount: p.span}
}
