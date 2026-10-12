package processing_test

import (
	"bytes"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exportvisualtopo"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strings"
	"testing"
)

func readVisualTopoDocument(t *testing.T, path string) source.Document {
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
func TestVisualTopoNativeSnapshotGoldens(t *testing.T) {
	files, e := filepath.Glob("../exportvisualtopo/testdata/circle*/*.txt")
	if e != nil || len(files) != 194 {
		t.Fatalf("goldens missing: %v", e)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			circle := 360
			if strings.Contains(path, "circle400") {
				circle = 400
			}
			name := strings.TrimSuffix(filepath.Base(path), ".txt")
			close := strings.HasSuffix(name, "-closed")
			name = strings.TrimSuffix(name, "-closed")
			var input string
			var templates []source.ReferencePrefix
			switch {
			case name == "features" || name == "extreme" || name == "dimensions" || strings.HasPrefix(name, "duplicates"):
				input = "../exportvisualtopo/testdata/" + name + ".top"
			case strings.HasPrefix(name, "geometry-"):
				input = "testdata/geometry/" + strings.TrimPrefix(name, "geometry-") + ".top"
			case strings.HasPrefix(name, "text-"):
				input = "../exporttext/testdata/" + strings.TrimPrefix(name, "text-") + ".top"
				if strings.HasPrefix(name, "text-api-") {
					input = "../top/testdata/" + strings.TrimPrefix(name, "text-") + ".top"
				}
			default:
				dir := "testdata/context/" + strings.TrimPrefix(name, "context-")
				if strings.HasPrefix(name, "nativecontext-") {
					dir = "../exportvisualtopo/testdata/" + strings.TrimPrefix(name, "nativecontext-")
				}
				input = dir + "/current.top"
				names, e := filepath.Glob(dir + "/*.top")
				if e != nil {
					t.Fatal(e)
				}
				for _, n := range names {
					if filepath.Base(n) == "current.top" {
						continue
					}
					b, e := os.ReadFile(n)
					if e != nil {
						t.Fatal(e)
					}
					p, e := top.ReadV3ReferencePrefix(b)
					if e != nil {
						t.Fatal(e)
					}
					templates = append(templates, p)
				}
			}
			d := readVisualTopoDocument(t, input)
			before := d.Bytes()
			r, e := exportvisualtopo.Build(d, templates, exportvisualtopo.Options{Circle: circle, Geometry: processing.GeometryOptions{CloseLoops: close}})
			if e != nil {
				t.Fatal(e)
			}
			want, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(r.Text, want) {
				os.MkdirAll("../../build/p11/diff", 0700)
				os.WriteFile("../../build/p11/diff/"+filepath.Base(path), r.Text, 0600)
				t.Fatalf("native bytes differ for %s (got %d want %d)", name, len(r.Text), len(want))
			}
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}
