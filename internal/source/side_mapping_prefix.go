package source

import "bytes"

type SideMappingPrefixOffsets struct {
	PlanElementsPrefixOffsets
	Side MappingOffsets
}

// SideMappingPrefix ends after a complete raw side mapping or an unsupported
// plan marker. SideMapping reports presence; an all-zero mapping is valid.
type SideMappingPrefix struct {
	PlanElementsPrefix
	side      Mapping
	present   bool
	prefix    []byte
	tailBytes int
}

func NewSideMappingPrefix(plan PlanElementsPrefix, side Mapping, present bool, prefix []byte, tailBytes int) SideMappingPrefix {
	return SideMappingPrefix{PlanElementsPrefix: plan, side: side, present: present, prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p SideMappingPrefix) SideMapping() (Mapping, bool) { return p.side, p.present }
func (p SideMappingPrefix) Bytes() []byte                { return bytes.Clone(p.prefix) }
func (p SideMappingPrefix) ConsumedOffset() int          { return len(p.prefix) }
func (p SideMappingPrefix) UnparsedTailSize() int        { return p.tailBytes }
func (p SideMappingPrefix) Offsets() SideMappingPrefixOffsets {
	return SideMappingPrefixOffsets{PlanElementsPrefixOffsets: p.PlanElementsPrefix.Offsets(), Side: p.side.Offsets()}
}
