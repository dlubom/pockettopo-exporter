package exportsurvex

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"pockettopo-exporter/internal/top"
)

func TestSurvexCompactMeasurementLayout(t *testing.T) {
	for _, zero := range []bool{false, true} {
		d := survexDocument(t, "testdata/features.top")
		if zero {
			raw := d.Bytes()
			binary.LittleEndian.PutUint16(raw[d.Trips()[0].Offsets().Declination.Start:], 0)
			var err error
			d, err = top.ReadV3(raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		r, err := Build(d)
		if err != nil {
			t.Fatal(err)
		}
		var declinations []string
		lines := strings.Split(string(r.Text), "\r\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "; measurement ") {
				t.Fatal("generated measurement label clutters output", line)
			}
			if strings.HasPrefix(line, "*declination ") {
				declinations = append(declinations, line)
			}
		}
		want := []string{"*declination 22.5 degrees", "*declination 0 degrees", "*declination 22.5 degrees"}
		if zero {
			want = []string{"*declination 0 degrees"}
		}
		if !reflect.DeepEqual(declinations, want) {
			t.Fatalf("declination transitions: got %q, want %q", declinations, want)
		}
		if !bytes.Contains(r.Text, []byte(`; comment "hello\r\n#Fix evil 0 0 0; \u017c"`)) {
			t.Fatal("source comment lost")
		}
		for _, row := range r.Report.Measurements {
			if !strings.HasPrefix(lines[row.Line-1], row.From.Survex+"\t"+row.To.Survex+"\t") {
				t.Fatal("measurement report points to wrong output line", row.Index)
			}
		}
	}
}
