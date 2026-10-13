package exportdxf3d

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func doc(t *testing.T, path string) source.Document {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestDXF3DNativeGoldens(t *testing.T) {
	b, e := os.ReadFile("testdata/cases.tsv")
	if e != nil {
		t.Fatal(e)
	}
	c := csv.NewReader(bytes.NewReader(b))
	c.Comma = '\t'
	rows, e := c.ReadAll()
	if e != nil || len(rows) != 167 {
		t.Fatal("manifest", len(rows), e)
	}
	for _, r := range rows[1:] {
		t.Run(r[0], func(t *testing.T) {
			want, e := os.ReadFile(filepath.Join("testdata", r[6]))
			if e != nil {
				t.Fatal(e)
			}
			if r[1] == "sort" {
				b, e := os.ReadFile(filepath.Join("testdata", r[2]))
				if e != nil {
					t.Fatal(e)
				}
				var stations []processing.GraphicsStation
				for i, line := range strings.Fields(string(b)) {
					raw, e := strconv.ParseUint(line, 10, 32)
					if e != nil {
						t.Fatal(e)
					}
					stations = append(stations, processing.GraphicsStation{Node: processing.GeometryNode{Key: strconv.Itoa(i), ID: source.NewStationID(uint32(raw))}})
				}
				w := writer{lim: defaultLimits}
				e = w.sort(stations, 0, len(stations)-1)
				if r[5] != "complete" {
					if e == nil {
						t.Fatal("native sort failure accepted")
					}
					return
				}
				if e != nil {
					t.Fatal(e)
				}
				var got strings.Builder
				for _, n := range stations {
					got.WriteString(n.Node.Key + "\r\n")
				}
				if got.String() != string(want) {
					t.Fatal("native sort order", got.String(), string(want))
				}
				return
			}
			if r[1] == "sequence" {
				b, e := os.ReadFile(filepath.Join("testdata", r[2]))
				if e != nil {
					t.Fatal(e)
				}
				w := writer{lim: defaultLimits, report: new(Report), lineNumber: 1}
				for _, name := range strings.Fields(string(b)) {
					data, e := os.ReadFile(filepath.Join("testdata", name))
					if e != nil {
						t.Fatal(e)
					}
					var pts []*point
					for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
						v := strings.Fields(line)
						var p [3]int32
						for i := range p {
							n, e := strconv.ParseInt(v[i], 10, 32)
							if e != nil {
								t.Fatal(e)
							}
							p[i] = int32(n)
						}
						pts = append([]*point{quantize(processing.Position{X: p[0], Y: p[1], H: p[2]})}, pts...)
					}
					if e = w.stretch(pts); e != nil {
						break
					}
				}
				if r[5] != "complete" {
					if e == nil {
						t.Fatal("sequence native failure accepted")
					}
					return
				}
				if e != nil || w.b.String() != string(want) {
					t.Fatal("persistent-normal native sequence", e, len(w.b.String()), len(want))
				}
				return
			}
			if r[1] == "model" {
				b, e := os.ReadFile(filepath.Join("testdata", r[2]))
				if e != nil {
					t.Fatal(e)
				}
				var pos []processing.Position
				for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
					cols := strings.Fields(line)
					var p [3]int32
					for i := range p {
						n, e := strconv.ParseInt(cols[i], 10, 32)
						if e != nil {
							t.Fatal(e)
						}
						p[i] = int32(n)
					}
					pos = append(pos, processing.Position{X: p[0], Y: p[1], H: p[2]})
				}
				w := writer{lim: limits{bytes: 1000000, entities: 1000, points: 1000, work: 10000}, report: new(Report), lineNumber: 1}
				var pts []*point
				for _, p := range pos {
					pts = append([]*point{quantize(p)}, pts...)
				}
				e = w.stretch(pts)
				if r[5] != "complete" {
					if e == nil {
						t.Fatal("unbounded native degeneracy accepted")
					}
					return
				}
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal([]byte(w.b.String()), want) {
					t.Fatal("model native bytes", len(w.b.String()), len(want))
				}
				return
			}
			d := doc(t, filepath.Join("../..", r[2]))
			before := d.Bytes()
			var templates []source.ReferencePrefix
			if r[3] == "True" {
				paths, e := filepath.Glob(filepath.Join("../..", filepath.Dir(r[2]), "*.top"))
				if e != nil {
					t.Fatal(e)
				}
				for _, p := range paths {
					if filepath.Base(p) == filepath.Base(r[2]) {
						continue
					}
					b, e := os.ReadFile(p)
					if e != nil {
						t.Fatal(e)
					}
					v, e := top.ReadV3ReferencePrefix(b)
					if e != nil {
						t.Fatal(e)
					}
					templates = append(templates, v)
				}
			}
			got, e := Build(d, templates, Options{Geometry: processing.GeometryOptions{CloseLoops: r[4] == "True"}})
			if r[5] != "complete" {
				if e == nil || len(got.DXF) != 0 {
					t.Fatal("native failure accepted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(got.DXF, want) {
				t.Fatal("full native bytes", len(got.DXF), len(want))
			}
			verifyReport(t, d, templates, got)
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestDXF3DLimits(t *testing.T) {
	d := doc(t, "testdata/fallback.top")
	opt := Options{}
	full, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	lim := limits{len(full.DXF), len(full.Report.Entities), len(full.Report.Points), 100000}
	if r, e := build(d, nil, opt, lim); e != nil || !bytes.Equal(r.DXF, full.DXF) {
		t.Fatal("inclusive budgets", e)
	}
	for _, l := range []limits{{lim.bytes - 1, lim.entities, lim.points, lim.work}, {lim.bytes, lim.entities - 1, lim.points, lim.work}, {lim.bytes, lim.entities, lim.points - 1, lim.work}, {lim.bytes, lim.entities, lim.points, 0}, {-1, 1, 1, 1}, {1, -1, 1, 1}, {1, 1, -1, 1}, {1, 1, 1, -1}} {
		r, e := build(d, nil, opt, l)
		if e == nil || len(r.DXF) != 0 || r.Report.Schema != 0 {
			t.Fatal("partial result on limit", l, e)
		}
	}
	// Find the precise work boundary and verify it is inclusive.
	lo, hi := 0, 100000
	for lo < hi {
		mid := (lo + hi) / 2
		l := lim
		l.work = mid
		if _, e := build(d, nil, opt, l); e != nil {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	lim.work = lo
	if r, e := build(d, nil, opt, lim); e != nil || !bytes.Equal(r.DXF, full.DXF) {
		t.Fatal("exact work limit", e)
	}
	lim.work--
	if r, e := build(d, nil, opt, lim); e == nil || len(r.DXF) > 0 {
		t.Fatal("one short work", e)
	}
	for _, o := range []Options{{Geometry: processing.GeometryOptions{MaxWork: -1}}, {Geometry: processing.GeometryOptions{MaxIterations: -1}}, {Geometry: processing.GeometryOptions{MaxWork: 1}}} {
		if r, e := Build(d, nil, o); e == nil || len(r.DXF) > 0 {
			t.Fatal("geometry bounds", e)
		}
	}
	if r, e := Build(source.Document{}, nil, opt); e == nil || len(r.DXF) > 0 {
		t.Fatal("unsupported")
	}
	empty := doc(t, "../processing/testdata/geometry/empty.top")
	if _, e := build(empty, nil, opt, limits{bytes: 102, work: 0}); e != nil {
		t.Fatal("zero work empty", e)
	}
	if _, e := build(empty, nil, opt, limits{bytes: 101}); e == nil {
		t.Fatal("footer budget")
	}
}
func TestDXF3DViewFailures(t *testing.T) {
	mk := func() *writer { return &writer{lim: defaultLimits, report: new(Report), lineNumber: 1} }
	station := func(key, start string, dist int32) processing.GraphicsStation {
		return processing.GraphicsStation{Node: processing.GeometryNode{Key: key, StartKey: start}, Distance: dist}
	}
	w := mk()
	v := processing.GraphicsGeometry{Actual: []processing.GraphicsStation{station("missing", "unknown", 1)}}
	if e := writeGeometry(w, v); e == nil || !strings.Contains(e.Error(), "start unknown") {
		t.Fatal("missing start", e)
	}
	w = mk()
	v.Actual = []processing.GraphicsStation{station("leg", "zero", 1), station("zero", "unknown", 0)}
	if e := writeGeometry(w, v); e == nil || !strings.Contains(e.Error(), "ancestor unknown") {
		t.Fatal("missing ancestor", e)
	}
	w = mk()
	v.Actual = []processing.GraphicsStation{station("no-start", "none", 1)}
	if e := writeGeometry(w, v); e != nil || w.report.Stretches[0].Omission != "no_start" {
		t.Fatal("no start", e)
	}
	w = mk()
	w.lim.work = 100
	v.Actual = []processing.GraphicsStation{station("leg", "zero", 1), station("zero", "zero", 0)}
	if e := writeGeometry(w, v); e == nil {
		t.Fatal("ancestor cycle work bound")
	}
	// Each prefix of a hull's work must return promptly and fail closed.
	pts := []*point{quantize(processing.Position{X: 1000}), quantize(processing.Position{Y: 1000}), quantize(processing.Position{H: 1000}), quantize(processing.Position{})}
	w = mk()
	if e := w.stretch(pts); e != nil {
		t.Fatal(e)
	}
	work := w.work
	for i := 0; i < work; i++ {
		w = mk()
		w.lim.work = i
		p := make([]*point, len(pts))
		for j, n := range pts {
			p[j] = &point{coordinate: n.coordinate}
		}
		if e := w.stretch(p); e == nil {
			t.Fatal("work prefix accepted", i)
		}
	}
	w = mk()
	w.lim.entities = 0
	if e := w.stretch(pts); e == nil {
		t.Fatal("hull entity limit")
	}
	w = mk()
	if e := w.stretch(nil); e != nil {
		t.Fatal(e)
	}
	// Identity equality, removal and error propagation do not use coordinates.
	w = mk()
	p, q := new(point), new(point)
	if !w.add(p, q) || w.add(p, q) || !w.remove(p, q) || w.remove(p, q) {
		t.Fatal("edge identity")
	}
	w.err = fmt.Errorf("sentinel")
	if w.tick() {
		t.Fatal("latched error")
	}
	w.emit("LINE", p, q)
}

func verifyReport(t *testing.T, d source.Document, templates []source.ReferencePrefix, result Result) {
	t.Helper()
	r := result.Report
	if r.Schema != 1 || r.Format != "pockettopo-1.372-3d-dxf" || r.Tool != "dev" || r.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || r.SourceSize != len(d.Bytes()) || r.Tail != d.TailKind() || r.OutputHash != fmt.Sprintf("%x", sha256.Sum256(result.DXF)) || r.OutputSize != len(result.DXF) || len(r.Losses) != 5 {
		t.Fatal("3D metadata")
	}
	mode := "isolated-input"
	if len(templates) > 0 {
		mode = "explicit-template-directory"
	}
	if r.Context != mode {
		t.Fatal("context")
	}
	rows := make([][]source.Measurement, len(templates)+1)
	refs := make([][]source.Reference, len(templates)+1)
	rows[len(templates)], refs[len(templates)] = d.Measurements(), d.References()
	rowCount, refCount := len(rows[len(templates)]), len(refs[len(templates)])
	for i, p := range templates {
		rows[i], refs[i] = p.Measurements(), p.References()
		rowCount += len(rows[i])
		refCount += len(refs[i])
	}
	if len(r.Rows) != rowCount || len(r.References) != refCount {
		t.Fatal("source cardinality")
	}
	seen := map[[2]int]bool{}
	for _, x := range r.Rows {
		k := [2]int{x.Source, x.Index}
		if seen[k] || x.Source < 0 || x.Source >= len(rows) || x.Index < 0 || x.Index >= len(rows[x.Source]) {
			t.Fatal("row identity", x)
		}
		seen[k] = true
		m := rows[x.Source][x.Index]
		if x.Span != m.Offsets().Record || x.From != m.From().Raw() || x.To != m.To().Raw() || x.Distance != m.DistanceMM() || x.Azimuth != m.AzimuthRaw() || x.Inclination != m.InclinationRaw() || x.Flags != m.FlagsRaw() || x.Roll != m.RollRaw() || x.Trip != m.TripIndexRaw() {
			t.Fatal("raw row", x)
		}
	}
	g, e := processing.GroupContext(d, templates)
	if e != nil {
		t.Fatal(e)
	}
	byKey := map[[2]int]int{}
	for i, x := range r.Rows {
		byKey[[2]int{x.Source, x.Index}] = i
	}
	for _, part := range []struct {
		groups  []processing.Group
		lineage []processing.ContextLineage
		prefix  string
	}{{g.Templates.Groups(), g.TemplateRows, "template:"}, {g.Current.Groups(), g.CurrentRows, ""}} {
		for _, group := range part.groups {
			for _, member := range group.Members() {
				l := part.lineage[member.RowIndex]
				x := r.Rows[byKey[[2]int{l.SourceIndex, l.RowIndex}]]
				if x.Group != fmt.Sprintf("%sgroup:%d", part.prefix, group.HeadIndex) || x.Reversed != member.Reversed {
					t.Fatal("group lineage")
				}
			}
		}
	}
	seen = map[[2]int]bool{}
	templateReference := 0
	for _, x := range r.References {
		k := [2]int{x.Source, x.Index}
		if seen[k] {
			t.Fatal("reference identity")
		}
		seen[k] = true
		v := refs[x.Source][x.Index]
		key := fmt.Sprintf("reference:%d", x.Index)
		if x.Source < len(templates) {
			key = fmt.Sprintf("template:reference:%d", templateReference)
			templateReference++
		}
		if x.Key != key || x.Span != v.Offsets().Record || x.RawID != v.Station().Raw() || x.East != v.EastMM() || x.North != v.NorthMM() || x.Altitude != v.AltitudeMM() {
			t.Fatal("raw reference", x)
		}
	}
	view, e := processing.PrepareGraphics(d, templates, r.Options.Geometry)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Geometry) != len(view.Positions) {
		t.Fatal("node count")
	}
	nodes := map[string]processing.GeometryNode{}
	for i, n := range view.Positions {
		x := r.Geometry[i]
		if x.Key != n.Key || x.Status != n.Status || x.StartKey != n.StartKey || x.ID.Raw != n.ID.Raw() || x.ID.Native != n.ID.NativeValue() || x.ID.Display != n.ID.String() || x.Span != n.SourceSpan || x.Position != n.Position || x.Vector != n.Vector || x.Correction != n.Correction {
			t.Fatal("node provenance")
		}
		nodes[n.Key] = n
	}
	for _, p := range r.Points {
		n, ok := nodes[p.Key]
		if !ok || p.Position != n.Position {
			t.Fatal("point node")
		}
		want := coordinate{(p.Position.X >> 4) << 4, ((-p.Position.Y) >> 4) << 4, ((-p.Position.H) >> 4) << 4}
		if p.Output != want {
			t.Fatal("quantized point")
		}
	}
	lines := strings.Split(string(result.DXF), "\r\n")
	end := 14
	for i, x := range r.Entities {
		count := 2
		if x.Kind == "3DFACE" {
			count = 4
		} else if x.Kind != "LINE" {
			t.Fatal("entity kind")
		}
		if x.FirstLine != end+1 || x.LastLine != x.FirstLine+4+6*count-1 || len(x.Points) != count {
			t.Fatal("entity line range", i, x)
		}
		if lines[x.FirstLine-1] != "0" || lines[x.FirstLine] != x.Kind || lines[x.FirstLine+1] != "8" || lines[x.FirstLine+2] != "0" {
			t.Fatal("entity header")
		}
		for j, p := range x.Points {
			if p < 0 || p >= len(r.Points) {
				t.Fatal("point index")
			}
			v := r.Points[p].Output
			for a, n := range []int32{v.X, v.Y, v.Z} {
				offset := x.FirstLine + 3 + 6*j + 2*a
				if lines[offset] != strconv.Itoa(10+10*a+j) || lines[offset+1] != strconv.FormatInt(int64(n), 10) {
					t.Fatal("entity coordinate")
				}
			}
		}
		if count == 4 && x.Points[2] != x.Points[3] {
			t.Fatal("face fourth vertex")
		}
		end = x.LastLine
	}
	if lines[end] != "0" || lines[end+1] != "ENDSEC" || lines[end+2] != "0" || lines[end+3] != "EOF" {
		t.Fatal("footer lineage")
	}
	expectedStations := map[string]processing.GraphicsStation{}
	for _, n := range view.Actual {
		if n.Node.Key != "reference:-1" {
			expectedStations[n.Node.Key] = n
		}
	}
	for _, n := range view.Template {
		expectedStations[n.Node.Key] = n
	}
	if len(r.Stretches) != len(expectedStations) {
		t.Fatal("native station inventory")
	}
	expectedOrder := []processing.GraphicsStation{}
	graphicNodes := map[string]processing.GraphicsStation{}
	for _, n := range view.Actual {
		graphicNodes[n.Node.Key] = n
		if n.Node.Key != "reference:-1" {
			expectedOrder = append(expectedOrder, n)
		}
	}
	for _, n := range view.Template {
		graphicNodes[n.Node.Key] = n
		expectedOrder = append(expectedOrder, n)
	}
	sortWriter := writer{lim: defaultLimits}
	if e := sortWriter.sort(expectedOrder, 0, len(expectedOrder)-1); e != nil {
		t.Fatal(e)
	}
	pointIndex, entityIndex := 0, 0
	for index, s := range r.Stretches {
		if s.Key != expectedOrder[index].Node.Key {
			t.Fatal("sorted stretch order")
		}
		n, ok := expectedStations[s.Key]
		if !ok || s.Start != n.Node.StartKey {
			t.Fatal("stretch node")
		}
		delete(expectedStations, s.Key)
		omission := ""
		if n.Distance <= 0 {
			omission = "nonpositive_distance"
		} else if n.Node.StartKey == "none" {
			omission = "no_start"
		} else if n.Flags&4 != 0 {
			omission = "invisible"
		}
		if s.Omission != omission {
			t.Fatal("native eligibility", s)
		}

		if s.FirstPoint != pointIndex || s.FirstEntity != entityIndex {
			t.Fatal("stretch ranges")
		}
		pointIndex += s.PointCount
		entityIndex += s.EntityCount
		if s.Omission != "" {
			if s.PointCount != 0 || s.EntityCount != 0 {
				t.Fatal("omitted geometry")
			}
			continue
		}
		if s.ResolvedStart == "" || s.ResolvedEnd == "" || s.PointCount < 2 || s.EntityCount < 1 {
			t.Fatal("stretch endpoints")
		}
		expectedStart := graphicNodes[s.Start]
		for len(expectedStart.Cross) == 0 && expectedStart.Distance == 0 && expectedStart.Node.StartKey != "none" {
			expectedStart = graphicNodes[expectedStart.Node.StartKey]
		}
		if s.ResolvedStart != expectedStart.Node.Key || s.ResolvedEnd != s.Key {
			t.Fatal("resolved endpoint lineage")
		}
		for _, e := range r.Entities[s.FirstEntity:entityIndex] {
			if e.Owner != s.Key {
				t.Fatal("entity owner")
			}
			for _, p := range e.Points {
				if p < s.FirstPoint || p >= pointIndex {
					t.Fatal("stretch point owner")
				}
			}
		}
	}
	if pointIndex != len(r.Points) || entityIndex != len(r.Entities) {
		t.Fatal("unowned geometry")
	}
}
func TestDXF3DDeterminismAndFallback(t *testing.T) {
	for _, name := range []string{"fallback", "zero-cross"} {
		d := doc(t, "testdata/"+name+".top")
		before := d.Bytes()
		a, e := Build(d, nil, Options{})
		if e != nil {
			t.Fatal(e)
		}
		b, e := Build(d, nil, Options{})
		if e != nil {
			t.Fatal(e)
		}
		x, _ := json.Marshal(a.Report)
		y, _ := json.Marshal(b.Report)
		if !bytes.Equal(x, y) || !bytes.Equal(a.DXF, b.DXF) || !bytes.Equal(before, d.Bytes()) {
			t.Fatal("determinism/source preservation")
		}
		verifyReport(t, d, nil, a)
		found := false
		for _, s := range a.Report.Stretches {
			if s.Start == "group:0" && s.Omission == "" {
				found = true
				expected := "reference:0"
				if name == "zero-cross" {
					expected = "group:0"
				}
				if s.ResolvedStart != expected {
					t.Fatal("native zero-distance fallback", name, s)
				}
			}
		}
		if !found {
			t.Fatal("fixture missing zero-distance semantic case")
		}
		hiddenRay := false
		for _, p := range a.Report.Points {
			if p.Key == "group:2" {
				hiddenRay = true
			}
		}
		if hiddenRay != (name == "fallback") {
			t.Fatal("invisible splay filtered")
		}
		// The returned result is detached; mutation cannot affect a later build.
		a.DXF[0] = 'x'
		a.Report.Points[0].Key = "wrong"
		next, e := Build(d, nil, Options{})
		if e != nil || !reflect.DeepEqual(next, b) {
			t.Fatal("result aliasing", e)
		}
	}
}

func TestDXF3DSingletonSortNeedsNoWork(t *testing.T) {
	// Native Array.Sort's strict range test returns for one station without
	// partitioning. A zero remaining budget must therefore accept this case.
	id := source.NewStationID(65536)
	values := []processing.GraphicsStation{{Node: processing.GeometryNode{ID: id}}}
	w := writer{lim: limits{work: 0}}
	if e := w.sort(values, 0, 0); e != nil || values[0].Node.ID != id {
		t.Fatal("native singleton sort charged partition work", e, values)
	}
}

func TestDXF3DSortWorkAndWrappedComparer(t *testing.T) {
	var values []processing.GraphicsStation
	for i := 0; i < 100; i++ {
		values = append(values, processing.GraphicsStation{Node: processing.GeometryNode{ID: source.NewStationID(uint32(65536 + (i*17)%11))}})
	}
	w := writer{lim: defaultLimits}
	if e := w.sort(append([]processing.GraphicsStation{}, values...), 0, len(values)-1); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < w.work; i++ {
		q := writer{lim: limits{work: i}}
		if e := q.sort(append([]processing.GraphicsStation{}, values...), 0, len(values)-1); e == nil {
			t.Fatal("sort work prefix", i)
		}
	}
	w.err = fmt.Errorf("sentinel")
	w.line("must not append")
	if w.b.Len() != 0 {
		t.Fatal("latched output error")
	}
}

func FuzzDXF3DHull(f *testing.F) {
	for _, b := range [][]byte{make([]byte, 48), []byte("tetrahedron with negative coordinates and quantization"), []byte("overflow cross products across distinct source point identities")} {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) < 24 || len(b) > 120 {
			return
		}
		build := func() ([]byte, string) {
			w := writer{lim: limits{bytes: 16384, entities: 32, work: 2000}, report: new(Report), lineNumber: 1}
			var pts []*point
			for i := 0; i+12 <= len(b); i += 12 {
				v := [3]int32{}
				for j := range v {
					k := i + j*4
					v[j] = int32(uint32(b[k]) | uint32(b[k+1])<<8 | uint32(b[k+2])<<16 | uint32(b[k+3])<<24)
				}
				pts = append(pts, quantize(processing.Position{X: v[0], Y: v[1], H: v[2]}))
			}
			e := w.stretch(pts)
			status := ""
			if e != nil {
				status = e.Error()
			}
			if w.work > 2000 || w.b.Len() > 16384 || len(w.report.Entities) > 32 {
				t.Fatal("hull bounds")
			}
			return []byte(w.b.String()), status
		}
		a, x := build()
		c, y := build()
		if !bytes.Equal(a, c) || x != y {
			t.Fatal("hull determinism")
		}
	})
}

func TestDXF3DWorkStateAndPartitionProgress(t *testing.T) {
	w := writer{lim: defaultLimits, work: -1}
	if w.tick() || w.err == nil {
		t.Fatal("negative consumed work must fail closed")
	}
	w = writer{lim: defaultLimits}
	if e := w.partitionProgress(0, 10, 0, 10); e == nil {
		t.Fatal("partition did not shrink")
	}
	w = writer{lim: limits{work: 0}}
	if e := w.partitionProgress(0, 10, 5, 4); e == nil {
		t.Fatal("partition work budget")
	}
}

func TestDXF3DNativeFaceIdentities(t *testing.T) {
	b, e := os.ReadFile("testdata/faces.tsv")
	if e != nil {
		t.Fatal(e)
	}
	c := csv.NewReader(bytes.NewReader(b))
	c.Comma = '\t'
	cases, e := c.ReadAll()
	if e != nil || len(cases) != 25 {
		t.Fatal("native face manifest", e, len(cases))
	}
	read := func(s string) coordinate {
		fields := strings.Split(s, ",")
		var v [3]int32
		for i := range v {
			n, e := strconv.ParseInt(fields[i], 10, 32)
			if e != nil {
				t.Fatal(e)
			}
			v[i] = int32(n)
		}
		return coordinate{v[0], v[1], v[2]}
	}
	for _, r := range cases[1:] {
		t.Run(r[0], func(t *testing.T) {
			a, b := new(point), new(point)
			a.coordinate = read(r[1])
			b.coordinate = read(r[2])
			var points []*point
			for _, p := range strings.Split(r[3], ";") {
				points = append(points, &point{coordinate: read(p)})
			}
			prev := read(r[4])
			w := writer{lim: defaultLimits, prev: vector{prev.X, prev.Y, prev.Z}}
			p := w.findFace(points, a, b)
			idx, e := strconv.Atoi(r[5])
			if e != nil {
				t.Fatal(e)
			}
			expected := read(r[6])
			if p != points[idx] || w.prev != (vector{expected.X, expected.Y, expected.Z}) || w.err != nil {
				t.Fatal("native point identity/normal", p, idx, w.prev, expected, w.err)
			}
		})
	}
}
