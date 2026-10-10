package top_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Expected rows come from independently constructed literal tables and pinned
// helper coordinates, never from the Go reader or reference probe output.
func TestPlanPolygonsNativeProbeTranscript(t *testing.T) {
	actual, e := os.ReadFile("testdata/native-plan-polygons-read.txt")
	if e != nil {
		t.Fatal(e)
	}
	var want strings.Builder
	write := func(s string) { want.WriteString(s + "\r\n") }
	pointText := func(pts [][2]int32) string {
		rows := make([]string, len(pts))
		for i, p := range pts {
			rows[i] = fmt.Sprintf("%d,%d", p[0], p[1])
		}
		return strings.Join(rows, ";")
	}
	native := func(name string, tables [][][2]int32, consumed, tail int) {
		rows := make([]string, len(tables))
		for i, pts := range tables {
			rows[i] = pointText(pts)
		}
		write(fmt.Sprintf("%s native polygons=%d points=%s consumed=%d tail=%d", name, len(tables), strings.Join(rows, "|"), consumed, tail))
	}
	bounded := func(name string, tables [][][2]int32, colors []byte, start int, stop byte, tail int) {
		rows := make([]string, len(tables))
		total := 0
		for i, pts := range tables {
			end := start + 6 + len(pts)*8
			rows[i] = fmt.Sprintf("%d:%d:%s:%d:%d", start, len(pts), pointText(pts), colors[i], end)
			start = end
			total += len(pts)
		}
		write(fmt.Sprintf("%s bounded polygons=%d total=%d records=%s marker=%d start=%d consumed=%d tail=%d", name, len(tables), total, strings.Join(rows, "|"), stop, start, start+1, tail))
	}
	fixture := [][][2]int32{}
	colors := []byte{}
	for i := 0; i < 7; i++ {
		x := int32(-6000 + 2000*i)
		fixture = append(fixture, [][2]int32{{x, -2000}, {x + 500, -1000}, {x + 1500, -1500}})
		colors = append(colors, byte(i+1))
	}
	fixture = append(fixture, [][2]int32{{-7500, 3500}})
	colors = append(colors, 1)
	bounded("fixture-full", fixture, colors, 146, 3, 309)
	bounded("fixture-exact", fixture, colors, 146, 3, 0)
	write("fixture-missing-stop bounded error=EndOfStreamException consumed=370 tail=0")
	native("fixture-terminated-full", fixture, 237, 309)
	native("fixture-terminated-exact", fixture, 237, 0)
	literal := func(n, c int) ([][][2]int32, []byte, int) {
		tables := make([][][2]int32, n)
		colors := make([]byte, n)
		size := 13
		for i := range tables {
			tables[i] = make([][2]int32, i%4)
			for j := range tables[i] {
				tables[i][j] = [2]int32{int32(i*66051 + j), -int32(i*16777217 + j)}
			}
			colors[i] = byte(i * 73)
			if c >= 0 {
				colors[i] = byte(c)
			}
			size += 6 + len(tables[i])*8
		}
		return tables, colors, size
	}
	for _, n := range []int{0, 1, 4, 17, 257} {
		tables, colors, size := literal(n, -1)
		native(fmt.Sprintf("literal-n=%d-tail", n), tables, size, 2)
		native(fmt.Sprintf("literal-n=%d-exact", n), tables, size, 0)
		bounded(fmt.Sprintf("literal-n=%d", n), tables, colors, 12, 0, 2)
	}
	for c := 0; c < 256; c++ {
		tables, colors, size := literal(4, c)
		name := fmt.Sprintf("color=%d", c)
		native(name, tables, size, 2)
		bounded(name, tables, colors, 12, 0, 2)
	}
	tables, colors, size := literal(5, -1)
	for m := 0; m < 256; m++ {
		if m != 1 {
			bounded(fmt.Sprintf("stop=%d", m), tables, colors, 12, byte(m), 2)
		}
	}
	for n := 12; n < size; n++ {
		write(fmt.Sprintf("cut=%d native error=EndOfStreamException consumed=%d tail=0", n, n))
		write(fmt.Sprintf("cut=%d bounded error=EndOfStreamException consumed=%d tail=0", n, n))
	}
	if !bytes.Equal(actual, []byte(want.String())) {
		t.Fatalf("native transcript mismatch: got hash %x, expected %x; %d/%d bytes", sha256.Sum256(actual), sha256.Sum256([]byte(want.String())), len(actual), want.Len())
	}
}
