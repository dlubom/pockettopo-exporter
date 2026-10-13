package processing

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

type polarShot struct {
	from, to                          uint32
	distance                          int32
	bearing, inclination, declination int16
}

func polarResult(t *testing.T, shots []polarShot) source.Document {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("Top\x03")
	put := func(v any) {
		t.Helper()
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	put(int32(len(shots)))
	for _, s := range shots {
		put(int64(0))
		put(byte(0))
		put(s.declination)
	}
	put(int32(len(shots)))
	for i, s := range shots {
		put(s.from)
		put(s.to)
		put(s.distance)
		put(s.bearing)
		put(s.inclination)
		put(uint16(0))
		put(int16(i))
	}
	put(int32(0))
	mapping := func() { put(int32(0)); put(int32(0)); put(int32(1000)) }
	mapping()
	mapping()
	put(byte(0))
	mapping()
	put(byte(0))
	put(int32(0))
	d, err := top.ReadV3(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAverageRepeatedLegsPolar(t *testing.T) {
	for _, c := range []struct {
		name                           string
		shots                          []polarShot
		distance, azimuth, inclination float64
	}{
		{"triple and reverse", []polarShot{{0, 1, 10000, 0, 0, 4096}, {1, 0, 12000, -32768, 0, 4096}, {0, 1, 14000, 0, 0, 4096}}, 12, 0, 0},
		{"north wrap", []polarShot{{0, 1, 10000, -64, 0, 0}, {0, 1, 10000, 64, 0, 0}}, 10 * math.Cos(math.Pi/512), 0, 0},
		{"different corrections", []polarShot{{0, 1, 10000, 0, 0, 0}, {0, 1, 10000, -16384, 0, 16384}}, 10, 0, 0},
		{"inclined vectors", []polarShot{{0, 1, 2000, 16384, 8192, 0}, {0, 1, 2000, 16384, -8192, 0}}, math.Sqrt(2), 90, 0},
		{"positive inclination", []polarShot{{0, 1, 2000, 16384, 8192, 0}, {0, 1, 2000, 16384, 8192, 0}}, 2, 90, 45},
		{"reversed vertical", []polarShot{{0, 1, 10000, 0, 16384, 0}, {1, 0, 12000, -32768, -16384, 0}}, 11, 0, 90},
		{"large safe sum", []polarShot{{0, 1, 2000000000, 0, 0, 0}, {0, 1, 2000000000, 0, 0, 0}}, 2000000, 0, 0},
		{"millimetre leg", []polarShot{{0, 1, 1, 0, 0, 0}, {0, 1, 1, 0, 0, 0}}, 0.001, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := polarResult(t, c.shots)
			g, err := AverageRepeatedLegs(r)
			if err != nil || len(g) != 1 {
				t.Fatal(g, err)
			}
			a := g[0]
			azError := math.Abs(math.Remainder(a.AzimuthDegrees-c.azimuth, 360))
			if math.Abs(a.DistanceM-c.distance) > 1e-9 || azError > 1e-9 || math.Abs(a.InclinationDegrees-c.inclination) > 1e-9 {
				t.Fatal(a)
			}
			if a.HeadIndex != 0 || len(a.Members) != len(c.shots) {
				t.Fatal(a)
			}
			for i, m := range a.Members {
				if m.RowIndex != i || m.Reversed != (c.shots[i].from == 1) {
					t.Fatal(m)
				}
			}
		})
	}
}

func TestAverageRepeatedLegsBoundaries(t *testing.T) {
	if _, err := AverageRepeatedLegs(source.Document{}); err == nil {
		t.Fatal("unsupported accepted")
	}
	shots := []polarShot{{0, 1, 10000, 0, 0, 0}, {0, 1, 0, 0, 0, 0}, {0, 1, 10000, 0, 0, 0}, {0, 1, 12000, 0, 0, 0}, {1, 0x80000000, 2000, 0, 0, 0}, {1, 0x80000000, 2000, 0, 0, 0}, {2, 3, 1000, 0, 0, 0}, {0, 1, 10000, 0, 0, 0}}
	g, err := AverageRepeatedLegs(polarResult(t, shots))
	if err != nil || len(g) != 1 || g[0].HeadIndex != 2 || len(g[0].Members) != 2 || g[0].DistanceM != 11 {
		t.Fatal(g, err)
	}
	if g, err = AverageRepeatedLegs(polarResult(t, nil)); err != nil || g == nil || len(g) != 0 {
		t.Fatal(g, err)
	}
	g, err = AverageRepeatedLegs(polarResult(t, []polarShot{{0, 1, 10000, 0, 0, 0}, {0, 1, 10000, -32768, 0, 0}}))
	if err == nil || g != nil {
		t.Fatal("contradictory readings became an equation", g, err)
	}
}

func TestAverageRepeatedLegsRoundedZeroBoundary(t *testing.T) {
	var shots []polarShot
	for i := 0; i < 999; i++ {
		shots = append(shots, polarShot{0, 1, 1, 0, 0, 0}, polarShot{0, 1, 1, -32768, 0, 0})
	}
	shots = append(shots, polarShot{0, 1, 2, 0, 0, 0}, polarShot{0, 1, 1, -32768, 0, 0})
	if _, err := AverageRepeatedLegs(polarResult(t, shots)); err == nil {
		t.Fatal("half-micrometre average rounds to a zero equation")
	}
}

func TestAverageRepeatedLegsMultipleGroups(t *testing.T) {
	shots := []polarShot{{0, 1, 10000, 0, 0, 0}, {0, 1, 12000, 0, 0, 0}, {1, 0x80000000, 2000, 0, 0, 0}, {1, 2, 2000, 0, 0, 0}, {1, 2, 4000, 0, 0, 0}, {2, 3, 0, 0, 0, 0}, {3, 4, 1000, 0, 0, 0}}
	g, err := AverageRepeatedLegs(polarResult(t, shots))
	if err != nil || len(g) != 2 || g[0].HeadIndex != 0 || g[1].HeadIndex != 3 || g[0].DistanceM != 11 || g[1].DistanceM != 3 {
		t.Fatal(g, err)
	}
}
