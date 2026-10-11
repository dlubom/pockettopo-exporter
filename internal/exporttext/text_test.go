package exporttext

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func TestTextNativeGoldens(t *testing.T) {
	for _, name := range []string{"empty", "mixed", "rounding", "flags", "numbers", "api-references", "api-drawings"} {
		t.Run(name, func(t *testing.T) {
			input := filepath.Join("testdata", name+".top")
			if name == "api-references" || name == "api-drawings" {
				input = filepath.Join("..", "top", "testdata", name+".top")
			}
			b, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			d, err := top.ReadV3(b)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("testdata", name+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := Build(d, name)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(result.Text, want) {
				t.Fatalf("native golden differs:\ngot %q\nwant %q", result.Text, want)
			}
			if !bytes.Equal(d.Bytes(), b) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestTextReportLineageLosses(t *testing.T) {
	b, err := os.ReadFile("testdata/mixed.top")
	if err != nil {
		t.Fatal(err)
	}
	d, err := top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d, "mixed")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Report
	if p.Schema != 1 || p.Tool != "dev" || p.Format != "pockettopo-1.372-text" || p.Context != "isolated-input" || p.Title != "mixed" || p.SourceHash != fmt.Sprintf("%x", sha256.Sum256(b)) || p.SourceSize != 536 || p.SourceVersion != 3 || p.Tail != source.TailZeroTrailer || p.TextHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || p.TextSize != 1186 {
		t.Fatalf("wrong provenance: %+v", p)
	}
	if p.Profile != "metres; circle=360; invariant culture; UTF-8 without BOM; CRLF" || !strings.Contains(p.Update, "Native Update geometry exceptions are not reproduced") {
		t.Fatalf("unreported context: %+v", p)
	}
	if len(p.Records) != 19 {
		t.Fatalf("lost source rows: %d", len(p.Records))
	}
	wantKinds := []string{"trip", "reference", "measurement"}
	start := 0
	for k, n := range []int{3, 5, 11} {
		for i := 0; i < n; i++ {
			record := p.Records[start+i]
			var span source.Span
			switch k {
			case 0:
				span = d.Trips()[i].Offsets().Record
			case 1:
				span = d.References()[i].Offsets().Record
			case 2:
				span = d.Measurements()[i].Offsets().Record
			}
			wantLine := []int{3, 7, 13}[k] + i
			if record.Kind != wantKinds[k] || record.Index != i || record.Span != span || record.Line != wantLine {
				t.Fatalf("lineage: %+v", record)
			}
		}
		start += n
	}
	first := p.Records[8].Raw.(measurementRaw)
	if first.From != (Identity{65536, 65536, "1.0"}) || first.To != (Identity{65537, 65537, "1.1"}) || first.Distance != 1000 || first.Azimuth != 100 || first.Inclination != 200 || first.Flags != 2 || first.Roll != 255 || first.Trip != 0 {
		t.Fatalf("raw measurement: %+v", first)
	}
	reserved := p.Records[14].Raw.(measurementRaw)
	if reserved.From.Native != -1 || reserved.To.Native != -2 || reserved.From.Raw == reserved.To.Raw || reserved.From.Name != "" || reserved.To.Name != "" {
		t.Fatalf("collapsed blank identity: %+v", reserved)
	}
	if raw := p.Records[0].Raw.(tripRaw); raw.Ticks != 638448479990000000 || raw.Declination != 1234 {
		t.Fatalf("raw trip: %+v", raw)
	}
	if raw := p.Records[3].Raw.(referenceRaw); raw.East != 1005 || raw.North != -1005 || raw.Altitude != 15 || raw.Station.Raw != 65536 {
		t.Fatalf("raw ref: %+v", raw)
	}
	wantLosses := map[string]int{"trip_date_and_declination": 3, "reference_precision": 5, "measurement_precision_and_metadata": 11, "comment_lf_removed": 3, "blank_station_identity": 8, "unresolved_trip_index": 2, "mappings_and_tail_omitted": 3}
	for _, loss := range p.Losses {
		if wantLosses[loss.Code] != loss.Count || loss.Detail == "" {
			t.Fatalf("wrong loss: %+v", loss)
		}
		delete(wantLosses, loss.Code)
	}
	if len(wantLosses) != 0 {
		t.Fatalf("missing losses: %v", wantLosses)
	}
	j1, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Build(d, "mixed")
	if err != nil {
		t.Fatal(err)
	}
	j2, err := json.Marshal(again.Report)
	if err != nil || !bytes.Equal(j1, j2) || !bytes.Equal(r.Text, again.Text) {
		t.Fatal("nondeterministic output")
	}
}

func TestTextRejectsUnsupportedAndBadTitle(t *testing.T) {
	b, err := os.ReadFile("testdata/empty.top")
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{b[:len(b)-1], append(append([]byte{}, b...), 1), append(append([]byte{}, b[:40]...), 255)} {
		d, err := top.ReadV3(data)
		if err != nil {
			continue
		}
		if r, err := Build(d, "test"); err == nil || len(r.Text) != 0 {
			t.Fatal("partial export accepted")
		}
	}
	if _, err := Build(source.Document{}, "test"); err == nil {
		t.Fatal("zero source accepted")
	}
	d, err := top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "bad\nname", "bad\tname", string([]byte{255})} {
		if _, err := Build(d, name); err == nil {
			t.Fatalf("bad title accepted: %q", name)
		}
	}
}

func TestTextDrawingLossAndNativeUpdateDeviation(t *testing.T) {
	b, err := os.ReadFile("../top/testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	d, err := top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d, "api-drawings")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range r.Report.Losses {
		if l.Code == "drawing_elements_omitted" {
			found = true
			if l.Count != 20 {
				t.Fatalf("lost drawing count: %+v", l)
			}
		}
	}
	if !found {
		t.Fatal("silent drawing omission")
	}
	b, err = os.ReadFile("testdata/update-error.top")
	if err != nil {
		t.Fatal(err)
	}
	d, err = top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	r, err = Build(d, "update-error")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(r.Text, []byte("= ")) || len(r.Report.Records) != 9 || !strings.Contains(r.Report.Update, "exceptions are not reproduced") {
		t.Fatal("native geometry failure deviation hidden")
	}
}

func TestTextOutputBudget(t *testing.T) {
	b, err := os.ReadFile("testdata/empty.top")
	if err != nil {
		t.Fatal(err)
	}
	d, err := top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{0, 23, 24, 25} {
		r, err := build(d, "empty", limit)
		if limit < 24 {
			if err == nil || len(r.Text) != 0 {
				t.Fatalf("budget %d accepted", limit)
			}
		} else if err != nil || len(r.Text) != 24 {
			t.Fatalf("inclusive budget %d rejected: %v", limit, err)
		}
	}
}

func TestTextMeasurementLossBoundaries(t *testing.T) {
	b, err := os.ReadFile("testdata/flags.top")
	if err != nil {
		t.Fatal(err)
	}
	d, err := top.ReadV3(b)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Build(d, "flags")
	if err != nil {
		t.Fatal(err)
	}
	assertLoss := func(r Result, code string, count int) {
		t.Helper()
		found := false
		for _, l := range r.Report.Losses {
			if l.Count <= 0 {
				t.Fatalf("empty loss entry: %+v", l)
			}
			if l.Code == code {
				found = true
				if l.Count != count {
					t.Fatalf("loss %s: %d, want %d", code, l.Count, count)
				}
			}
		}
		if !found {
			t.Fatalf("missing loss %s", code)
		}
	}
	// All 256 flag bytes are present. Exactly 128 carry a comment with one LF;
	// the unchanged trip/reference tables contain three more LF characters.
	assertLoss(r, "comment_lf_removed", 131)
	b, err = os.ReadFile("testdata/empty.top")
	if err != nil {
		t.Fatal(err)
	}
	// Insert one actual serialized measurement with trip index zero, while the
	// trip table stays empty. Zero is already out of range in this document.
	shot := make([]byte, 20)
	binary.LittleEndian.PutUint32(shot[0:4], 1)
	binary.LittleEndian.PutUint32(shot[4:8], 2)
	binary.LittleEndian.PutUint32(shot[8:12], 1000)
	data := append(bytes.Clone(b[:12]), shot...)
	data = append(data, b[12:]...)
	binary.LittleEndian.PutUint32(data[8:12], 1)
	d, err = top.ReadV3(data)
	if err != nil {
		t.Fatal(err)
	}
	r, err = Build(d, "no-trips")
	if err != nil || !bytes.Contains(r.Text, []byte("[1]")) {
		t.Fatalf("unresolved trip text: %q, %v", r.Text, err)
	}
	assertLoss(r, "unresolved_trip_index", 1)
}
