package top_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Expectations use literal inputs and independently pinned helper offsets,
// never the Go reader or the native transcript to generate expected values.
func TestSideMappingPrefixOriginalMethodTranscript(t *testing.T) {
	var rows []string
	for _, pixels := range []int32{5, 10} {
		success := func(name string, start int, x, y, scale int32, tail int) {
			rows = append(rows, fmt.Sprintf("%s start=%d raw=%d,%d,%d native=%d,%d,%d pixels=%d consumed=%d tail=%d", name, start, x, y, scale, x, y, scale/pixels, pixels, start+12, tail))
		}
		for _, n := range []int{0, 1, 4, 17} {
			start := 13 + ((n+1)/2)*14 + (n/2)*17
			success(fmt.Sprintf("many=%d-tail", n), start, -66051, 16777217, -501, 2)
			success(fmt.Sprintf("many=%d-exact", n), start, -66051, 16777217, -501, 0)
			for cut := 0; cut < 12; cut++ {
				rows = append(rows, fmt.Sprintf("many=%d-cut=%d start=%d error=EndOfStreamException pixels=%d consumed=%d tail=0", n, cut, start, pixels, start+cut))
			}
		}
		for field := 0; field < 3; field++ {
			for _, value := range []int32{-2147483648, -16777217, -501, -11, -5, -1, 0, 1, 4, 5, 9, 10, 501, 16777217, 2147483647} {
				v := [3]int32{-66051, 16777217, -501}
				v[field] = value
				success(fmt.Sprintf("field=%d value=%d", field, value), 75, v[0], v[1], v[2], 2)
			}
		}
		success("references-full", 231, 0, 0, 500, 5)
		success("references-exact", 231, 0, 0, 500, 0)
		success("drawings-full", 405, 300, -400, 500, 263)
		success("drawings-exact", 405, 300, -400, 500, 0)
	}
	got, e := os.ReadFile("testdata/native-side-mapping-read.txt")
	if e != nil {
		t.Fatal(e)
	}
	want := []byte(strings.Join(rows, "\r\n") + "\r\n")
	if len(rows) != 210 || !bytes.Equal(got, want) {
		t.Fatalf("native transcript differs: got %d bytes, want %d", len(got), len(want))
	}
}
