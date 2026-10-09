package source_test

import (
	"fmt"
	"testing"

	"pockettopo-exporter/internal/source"
)

// Source-derived expectations: PocketTopo 1.372 ID.cs Read/ToString and IL
// Read RVA 0x22348, ToString RVA 0x22070. These are not Go-generated goldens.
// See README for reference hashes and the separately captured native evidence.
func TestStationIDBoundaries(t *testing.T) {
	cases := []struct {
		raw   uint32
		value int32
		text  string
	}{
		{0x7fffffff, 2147483647, "32767.65535"},
		{0x80000000, -1, ""},
		{0x80000001, -1048576, "0"},
		{0x80000002, -1048575, "1"},
		{0x800ffeff, -258, "1048318"},
		{0x800fff00, -257, "1048319"},
		{0x800fff01, -256, ""},
		{0x800fff02, -255, ""},
		{0x800ffffe, -3, ""},
		{0x800fffff, -2, ""},
		{0x80100000, -1, ""},
		{0x80100001, 0, "0.0"},
		{0x80100002, 1, "0.1"},
		{0xffffffff, 2146435070, "32751.65534"},
		{0x00000000, 0, "0.0"},
		{0x00000001, 1, "0.1"},
		{0x0000ffff, 65535, "0.65535"},
		{0x00010000, 65536, "1.0"},
		{0x00010001, 65537, "1.1"},
		{0x00010009, 65545, "1.9"},
		{0x7feffffe, 2146435070, "32751.65534"},
		{0x7fefffff, 2146435071, "32751.65535"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("raw_%08x", tc.raw), func(t *testing.T) {
			id := source.NewStationID(tc.raw)
			for range 2 {
				if got := id.NativeValue(); got != tc.value {
					t.Fatalf("NativeValue = %d, want %d", got, tc.value)
				}
				if got := id.String(); got != tc.text {
					t.Fatalf("String = %q, want %q", got, tc.text)
				}
				if got := id.Raw(); got != tc.raw {
					t.Fatalf("Raw after decoding/formatting = %08x, want %08x", got, tc.raw)
				}
			}
		})
	}
}

// Independently recorded reflection against the original 1.372 assembly on
// 2026-10-08: JKTZ issue #135. This reuses those five observations, without
// claiming a fresh native run or that the entire boundary table was captured.
func TestStationIDRecordedNativeProbe(t *testing.T) {
	cases := []struct {
		raw   uint32
		value int32
		text  string
	}{
		{0x80000000, -1, ""},
		{0x80000001, -1048576, "0"},
		{0x800fff00, -257, "1048319"},
		{0x800fffff, -2, ""},
		{0x00010009, 65545, "1.9"},
	}
	for _, tc := range cases {
		id := source.NewStationID(tc.raw)
		if id.Raw() != tc.raw || id.NativeValue() != tc.value || id.String() != tc.text {
			t.Errorf("native probe %08x: raw=%08x internal=%d text=%q", tc.raw, id.Raw(), id.NativeValue(), id.String())
		}
	}
}

func TestStationIDIdentity(t *testing.T) {
	cases := []struct {
		name  string
		a, b  uint32
		equal bool
	}{
		{"same raw", 0x00010009, 0x00010009, true},
		{"zero alias", 0x00000000, 0x80100001, true},
		{"minor alias", 0x00000001, 0x80100002, true},
		{"maximum alias", 0x7feffffe, 0xffffffff, true},
		{"undefined alias", 0x80000000, 0x80100000, true},
		{"empty but distinct", 0x80000000, 0x800fffff, false},
		{"reserved neighbors", 0x800fff01, 0x800fff02, false},
		{"plain zero versus major zero", 0x80000001, 0x00000000, false},
		{"major difference", 0x00010001, 0x00020001, false},
		{"minor difference", 0x00010001, 0x00010002, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := source.NewStationID(tc.a), source.NewStationID(tc.b)
			if a.SameIdentity(b) != tc.equal || b.SameIdentity(a) != tc.equal {
				t.Fatalf("SameIdentity(%08x, %08x), want %t", tc.a, tc.b, tc.equal)
			}
			if a.Raw() != tc.a || b.Raw() != tc.b {
				t.Fatal("identity comparison changed source bits")
			}
			if tc.a != tc.b && a == b {
				t.Fatal("distinct raw records compare equal with Go ==")
			}
		})
	}
}

func TestStationIDReservedRange(t *testing.T) {
	// Every value -256..-1 is empty, but only internal equality collapses IDs.
	undefined := source.NewStationID(0x80000000)
	for raw := uint32(0x800fff01); raw <= 0x80100000; raw++ {
		id := source.NewStationID(raw)
		want := int32(int64(raw) - 4294967296 + 2146435071)
		if id.NativeValue() != want || id.String() != "" || id.Raw() != raw {
			t.Fatalf("reserved %08x: raw=%08x internal=%d text=%q", raw, id.Raw(), id.NativeValue(), id.String())
		}
		if id.SameIdentity(undefined) != (want == -1) {
			t.Fatalf("reserved %08x conflated with UNDEF", raw)
		}
		if raw < 0x80100000 && id.SameIdentity(source.NewStationID(raw+1)) {
			t.Fatalf("reserved neighbors %08x and %08x compare equal", raw, raw+1)
		}
	}
}

func TestStationIDZeroValue(t *testing.T) {
	var id source.StationID
	if id.Raw() != 0 || id.NativeValue() != 0 || id.String() != "0.0" || !id.SameIdentity(source.NewStationID(0)) {
		t.Fatal("zero value must represent raw 0, not UNDEF")
	}
}
