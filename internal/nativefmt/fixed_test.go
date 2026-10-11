package nativefmt

import (
	"testing"
)

// Independent .NET 2.0 f-format observations from the original assembly;
// catches banker rounding, binary-tail rounding and lost significant precision.
func TestFixedNative(t *testing.T) {
	for _, tc := range []struct {
		value  float64
		digits int
		want   string
	}{
		{1.005, 2, "1.01"}, {-1.005, 2, "-1.01"}, {0.005, 2, "0.01"},
		{0.125, 2, "0.13"}, {-0.125, 2, "-0.13"}, {0, 3, "0.000"},
		{-0.001, 2, "0.00"}, {0.00001, 2, "0.00"}, {9.999, 2, "10.00"},
		{9223372036854775807.0 / 1000, 2, "9223372036854780.00"},
		{123456789012345.0 / 1000, 2, "123456789012.35"},
		{2147483647.0 / 1000, 3, "2147483.647"},
		{4995643414436645, 2, "4995643414436650.00"},
		{-5769937737102365, 2, "-5769937737102370.00"},
		{65535.0 * 360 / 65536, 2, "359.99"}, {-32768.0 * 360 / 65536, 2, "-180.00"},
	} {
		if got := Fixed(tc.value, tc.digits); got != tc.want {
			t.Errorf("Fixed(%g,%d)=%q want %q", tc.value, tc.digits, got, tc.want)
		}
	}
}

func TestFixedDecimalBoundaries(t *testing.T) {
	// Exact powers and integers have unambiguous fixed representations, including
	// the 15-significant-digit boundary and the documented zero-place case.
	if got := Fixed(1000000000000, 2); got != "1000000000000.00" {
		t.Fatalf("significant-digit boundary: %q", got)
	}
	if got := Fixed(12, 0); got != "12" {
		t.Fatalf("zero decimal places: %q", got)
	}
}
