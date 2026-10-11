package source

import "bytes"

type SideMarkerPrefixOffsets struct {
	SideMappingPrefixOffsets
	SideMarker Span
}

// SideMarkerPrefix ends after one raw side marker, or an unsupported plan stop.
// SideMarkerRaw reports presence independently of the byte value.
type SideMarkerPrefix struct {
	SideMappingPrefix
	marker    byte
	present   bool
	span      Span
	prefix    []byte
	tailBytes int
}

func NewSideMarkerPrefix(side SideMappingPrefix, marker byte, present bool, span Span, prefix []byte, tailBytes int) SideMarkerPrefix {
	return SideMarkerPrefix{SideMappingPrefix: side, marker: marker, present: present, span: span, prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p SideMarkerPrefix) SideMarkerRaw() (byte, bool) { return p.marker, p.present }
func (p SideMarkerPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p SideMarkerPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p SideMarkerPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p SideMarkerPrefix) Offsets() SideMarkerPrefixOffsets {
	return SideMarkerPrefixOffsets{SideMappingPrefixOffsets: p.SideMappingPrefix.Offsets(), SideMarker: p.span}
}
