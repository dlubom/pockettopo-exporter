package source

import "bytes"

type PlanMappingPrefixOffsets struct {
	OverviewPrefixOffsets
	Plan MappingOffsets
}

// PlanMappingPrefix ends after the plan/outline mapping, before the first
// element marker. The remaining drawing bytes may be absent or arbitrary.
type PlanMappingPrefix struct {
	overview  OverviewPrefix
	plan      Mapping
	prefix    []byte
	tailBytes int
}

func NewPlanMappingPrefix(overview OverviewPrefix, plan Mapping, prefix []byte, tailBytes int) PlanMappingPrefix {
	return PlanMappingPrefix{overview: overview, plan: plan,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}

func (p PlanMappingPrefix) Header() [4]byte             { return p.overview.Header() }
func (p PlanMappingPrefix) Version() byte               { return p.overview.Version() }
func (p PlanMappingPrefix) TripCountRaw() int32         { return p.overview.TripCountRaw() }
func (p PlanMappingPrefix) Trips() []Trip               { return p.overview.Trips() }
func (p PlanMappingPrefix) MeasurementCountRaw() int32  { return p.overview.MeasurementCountRaw() }
func (p PlanMappingPrefix) Measurements() []Measurement { return p.overview.Measurements() }
func (p PlanMappingPrefix) ReferenceCountRaw() int32    { return p.overview.ReferenceCountRaw() }
func (p PlanMappingPrefix) References() []Reference     { return p.overview.References() }
func (p PlanMappingPrefix) OverviewMapping() Mapping    { return p.overview.OverviewMapping() }
func (p PlanMappingPrefix) PlanMapping() Mapping        { return p.plan }
func (p PlanMappingPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanMappingPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanMappingPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanMappingPrefix) Offsets() PlanMappingPrefixOffsets {
	return PlanMappingPrefixOffsets{OverviewPrefixOffsets: p.overview.Offsets(), Plan: p.plan.Offsets()}
}
