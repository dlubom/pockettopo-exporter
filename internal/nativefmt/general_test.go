package nativefmt

import (
	"math"
	"testing"
)

func TestGeneralNative(t *testing.T) {
	for _, tc := range []struct {
		v    float64
		want string
	}{{0, "0"}, {math.Copysign(0, -1), "0"}, {1.0 / 3, "0.333333333333333"}, {0.00001, "1E-05"}, {0.0001, "0.0001"}, {1000000000000000, "1E+15"}, {-9223372036854775808.0, "-9.22337203685478E+18"}, {0.1, "0.1"}} {
		if got := General(tc.v); got != tc.want {
			t.Errorf("General(%g)=%s want%s", tc.v, got, tc.want)
		}
	}
}
