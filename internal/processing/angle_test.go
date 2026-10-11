package processing

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestNativeRectGolden(t *testing.T) {
	b, err := os.ReadFile("testdata/rect.tsv")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.FieldsFunc(string(b), func(r rune) bool { return r == '\r' || r == '\n' }) {
		var length, angle, x, y int32
		if n, e := fmt.Sscanf(line, "%d\t%d\t%d\t%d", &length, &angle, &x, &y); e != nil || n != 4 {
			t.Fatal(line, e)
		}
		gotX, gotY := toRect(length, angle)
		if gotX != x || gotY != y {
			t.Fatalf("ToRect(%d,%d)=(%d,%d), native=(%d,%d)", length, angle, gotX, gotY, x, y)
		}
	}
}
func TestNativeSineTable(t *testing.T) {
	b, e := os.ReadFile("testdata/sine-table.bin")
	if e != nil {
		t.Fatal(e)
	}
	if len(b) != 16385*4 {
		t.Fatal(len(b))
	}
	for i := 0; i <= 16384; i++ {
		if got := sineTable[i]; got != int32(binary.LittleEndian.Uint32(b[i*4:])) {
			t.Fatalf("native sine[%d]: %d", i, got)
		}
	}
}
