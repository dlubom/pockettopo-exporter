package source_test

import (
	"bytes"
	"pockettopo-exporter/internal/source"
	"testing"
)

func TestSideMarkerPrefixConstructorCopies(t *testing.T) {
	raw := []byte{1, 2, 0}
	side := source.NewSideMappingPrefix(source.PlanElementsPrefix{}, source.Mapping{}, true, []byte{1, 2}, 99)
	span := source.Span{Start: 2, End: 3}
	p := source.NewSideMarkerPrefix(side, 0, true, span, raw, 7)
	raw[0] = 255
	out := p.Bytes()
	out[1] = 255
	offsets := p.Offsets()
	offsets.SideMarker.Start = 0
	m, ok := p.SideMarkerRaw()
	if !ok || m != 0 || !bytes.Equal(p.Bytes(), []byte{1, 2, 0}) || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 7 || p.Offsets().SideMarker != span || p.Offsets().SideMappingPrefixOffsets != side.Offsets() || p.SideMappingPrefix.ConsumedOffset() != 2 {
		t.Fatal("lost metadata or alias")
	}
	var zero source.SideMarkerPrefix
	m, ok = zero.SideMarkerRaw()
	if ok || m != 0 || zero.ConsumedOffset() != 0 || zero.UnparsedTailSize() != 0 || zero.Offsets().SideMarker != (source.Span{}) || zero.Offsets().SideMappingPrefixOffsets != (source.SideMappingPrefix{}).Offsets() {
		t.Fatal("zero value")
	}
}
