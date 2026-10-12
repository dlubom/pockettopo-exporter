package nativefmt

import (
	"math"
	"strconv"
	"strings"
)

// General reproduces Framework's invariant default Double.ToString profile:
// 15 significant digits, trimmed fraction and uppercase two-digit exponent.
// Callers supply finite values derived from bounded native integers and scale.
func General(value float64) string {
	if value == 0 {
		return "0"
	}
	s := strconv.FormatFloat(math.Abs(value), 'e', 14, 64)
	exponent, _ := strconv.Atoi(s[strings.IndexByte(s, 'e')+1:])
	// Fixed performs Framework's rounding of the original binary value.
	places := 14 - exponent
	if places < 0 {
		places = 0
	}
	decimal := Fixed(value, places)
	negative := strings.HasPrefix(decimal, "-")
	digits := strings.ReplaceAll(strings.TrimPrefix(decimal, "-"), ".", "")
	digits = strings.TrimLeft(digits, "0")
	if exponent < 0 {
		exponent = len(digits) - places - 1
	} else {
		exponent = len(strings.Split(strings.TrimPrefix(decimal, "-"), ".")[0]) - 1
	}
	digits = strings.TrimRight(digits, "0")
	if exponent >= 15 || exponent < -4 {
		mantissa := digits[:1]
		if len(digits) > 1 {
			mantissa = strings.Join([]string{mantissa, ".", digits[1:]}, "")
		}
		sign := "+"
		if exponent < 0 {
			sign = "-"
		}
		e := strconv.Itoa(abs(exponent))
		if len(e) < 2 {
			e = strings.Join([]string{"0", e}, "")
		}
		if negative {
			mantissa = strings.Join([]string{"-", mantissa}, "")
		}
		return strings.Join([]string{mantissa, "E", sign, e}, "")
	}
	if strings.Contains(decimal, ".") {
		decimal = strings.TrimRight(strings.TrimRight(decimal, "0"), ".")
	}
	return decimal
}
