package source

import "bytes"

type PlanMarkerPrefixOffsets struct {
	PlanMappingPrefixOffsets
	Marker Span
}

// PlanMarkerPrefix ends after the first plan/outline element kind byte.
// It preserves every marker, including 0, without reading any payload.
type PlanMarkerPrefix struct {
	plan      PlanMappingPrefix
	marker    byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanMarkerPrefix(plan PlanMappingPrefix, marker byte, span Span, prefix []byte, tailBytes int) PlanMarkerPrefix {
	return PlanMarkerPrefix{plan: plan, marker: marker, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}

func (p PlanMarkerPrefix) Header() [4]byte             { return p.plan.Header() }
func (p PlanMarkerPrefix) Version() byte               { return p.plan.Version() }
func (p PlanMarkerPrefix) TripCountRaw() int32         { return p.plan.TripCountRaw() }
func (p PlanMarkerPrefix) Trips() []Trip               { return p.plan.Trips() }
func (p PlanMarkerPrefix) MeasurementCountRaw() int32  { return p.plan.MeasurementCountRaw() }
func (p PlanMarkerPrefix) Measurements() []Measurement { return p.plan.Measurements() }
func (p PlanMarkerPrefix) ReferenceCountRaw() int32    { return p.plan.ReferenceCountRaw() }
func (p PlanMarkerPrefix) References() []Reference     { return p.plan.References() }
func (p PlanMarkerPrefix) OverviewMapping() Mapping    { return p.plan.OverviewMapping() }
func (p PlanMarkerPrefix) PlanMapping() Mapping        { return p.plan.PlanMapping() }
func (p PlanMarkerPrefix) MarkerRaw() byte             { return p.marker }
func (p PlanMarkerPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanMarkerPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanMarkerPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanMarkerPrefix) Offsets() PlanMarkerPrefixOffsets {
	return PlanMarkerPrefixOffsets{PlanMappingPrefixOffsets: p.plan.Offsets(), Marker: p.span}
}
