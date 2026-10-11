package source_test

import (
	"bytes"
	"pockettopo-exporter/internal/source"
	"testing"
)

func TestSideMappingPrefixConstructorCopies(t *testing.T) {
	raw := []byte{1, 2, 3}
	plan := source.NewPlanElementsPrefix(source.PlanMappingPrefix{}, nil, 0, source.Span{Start: 1, End: 2}, source.PlanStopTerminated, 0, []byte{1, 2}, 99)
	offsets := source.MappingOffsets{Record: source.Span{Start: 2, End: 14}, X0: source.Span{Start: 2, End: 6}, Y0: source.Span{Start: 6, End: 10}, Scale: source.Span{Start: 10, End: 14}}
	m := source.NewMapping(-1, 2, 0, offsets)
	p := source.NewSideMappingPrefix(plan, m, true, raw, 7)
	raw[0] = 255
	out := p.Bytes()
	out[1] = 255
	copyOffsets := p.Offsets()
	copyOffsets.Side.X0.Start = 0
	got, ok := p.SideMapping()
	if !ok || got != m || !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 7 || p.Offsets().Side != offsets || p.Offsets().PlanElementsPrefixOffsets != plan.Offsets() || p.StopReason() != source.PlanStopTerminated || p.PlanElementsPrefix.ConsumedOffset() != 2 {
		t.Fatal("lost data or alias")
	}
	var zero source.SideMappingPrefix
	m, ok = zero.SideMapping()
	if ok || m != (source.Mapping{}) || zero.ConsumedOffset() != 0 || zero.UnparsedTailSize() != 0 {
		t.Fatal("zero value")
	}
}
