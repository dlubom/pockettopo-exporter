package exportwalls

import (
	"encoding/binary"
	"strings"
	"testing"

	"pockettopo-exporter/internal/top"
)

func TestWallsAveragesRepeatedLegs(t *testing.T) {
	d := wallsDocument(t, "testdata/features.top")
	raw := d.Bytes()
	rows := d.Measurements()
	// Three real observations of one leg: forward 10m, reverse 12m, forward 14m.
	binary.LittleEndian.PutUint32(raw[rows[1].Offsets().Record.Start+8:], 12000)
	at := rows[2].Offsets().Record.Start
	copy(raw[at:at+8], raw[rows[0].Offsets().Record.Start:rows[0].Offsets().Record.Start+8])
	binary.LittleEndian.PutUint32(raw[at+8:], 14000)
	binary.LittleEndian.PutUint16(raw[at+12:], 0)
	binary.LittleEndian.PutUint16(raw[at+14:], 0)
	binary.LittleEndian.PutUint16(raw[at+18:], 0)
	d, err := top.ReadV3(raw)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d)
	if err != nil {
		t.Fatal(err)
	}
	var data []string
	for _, line := range strings.Split(string(r.Text), "\r\n") {
		if line != "" && line[0] != ';' && line[0] != '#' {
			data = append(data, line)
		}
	}
	if len(data) != 3 || data[0] != "0.0\t0.1\t12\t0\t0" {
		t.Fatalf("triple must become one averaged leg; got %q", data)
	}
	if len(r.Report.Averages) != 1 || len(r.Report.Averages[0].Members) != 3 || !r.Report.Averages[0].Members[1].Reversed || r.Report.Averages[0].Line != r.Report.Measurements[0].Line {
		t.Fatal("average lineage", r.Report.Averages)
	}
	if len(r.Report.Measurements) != 5 {
		t.Fatal("raw observations lost")
	}
	for i := 0; i < 3; i++ {
		if r.Report.Measurements[i].Line != r.Report.Measurements[0].Line || r.Report.Measurements[i].Distance != int32(10000+2000*i) {
			t.Fatal("lost raw reading or averaged line ownership", i)
		}
	}
}

func TestWallsAverageCancellation(t *testing.T) {
	d := wallsDocument(t, "testdata/features.top")
	raw := d.Bytes()
	rows := d.Measurements()
	at := rows[1].Offsets().Record.Start
	copy(raw[at:at+8], raw[rows[0].Offsets().Record.Start:rows[0].Offsets().Record.Start+8])
	d, err := top.ReadV3(raw)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d)
	if err == nil || !strings.Contains(err.Error(), "cancel to zero") || len(r.Text) != 0 {
		t.Fatal(r, err)
	}
}

func TestWallsMixedDeclinationAverage(t *testing.T) {
	d := wallsDocument(t, "testdata/features.top")
	raw := d.Bytes()
	at := d.Measurements()[1].Offsets().Record.Start
	// Reverse observation has Auto/zero correction and a 202.5 degree bearing.
	// The head uses bearing 0 plus explicit 22.5 degrees: both describe one vector.
	binary.LittleEndian.PutUint16(raw[at+12:], 36864)
	binary.LittleEndian.PutUint16(raw[at+18:], 1)
	d, err := top.ReadV3(raw)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d)
	if err != nil {
		t.Fatal(err)
	}
	text := string(r.Text)
	if !strings.Contains(text, "#Units Decl=22.5\r\n0.0\t0.1\t10\t0\t0\r\n#Units Decl=0\r\n0.1\t0.2") || r.Report.Measurements[1].Correction != 0 || r.Report.Measurements[0].Line != r.Report.Measurements[1].Line {
		t.Fatal(text, r.Report.Measurements)
	}
}
