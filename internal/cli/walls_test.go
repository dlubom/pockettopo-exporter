package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exportwalls"
	"strings"
	"testing"
)

func TestWallsCLIExport(t *testing.T) {
	var out, err bytes.Buffer
	dir := filepath.Join(t.TempDir(), "out")
	args := []string{"export", "../exportwalls/testdata/features.top", "--format", "walls", "--output", dir}
	if Run(args, &out, &err) != 0 {
		t.Fatal("Walls export:", err.String())
	}
	got, e := os.ReadFile(filepath.Join(dir, "survey.srv"))
	if e != nil {
		t.Fatal(e)
	}
	want, e := os.ReadFile("../exportwalls/testdata/features-averaged.srv")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Walls golden: %s", got)
	}
	if out.String() != "Exported survey.srv and report.json\n" {
		t.Fatal("notification", out.String())
	}
	if Run(args, &out, &err) != 1 {
		t.Fatal("replaced existing output")
	}
}

func TestWallsCLIOptionsAndFailures(t *testing.T) {
	input := "../exportwalls/testdata/features.top"
	for _, args := range [][]string{
		{"export", input, "--format", "walls"}, {"export", "", "--format", "walls", "--output", "x"},
		{"export", "--bad", "--format", "walls", "--output", "x"}, {"export", input, "--format", "walls", "bad", "x"},
		{"export", input, "--format", "walls", "--output", ""}, {"export", input, "--format", "walls", "--output", "x", "--close-loops"},
		{"export", input, "--format", "walls", "--output", "x", "--template-directory", "."},
	} {
		var out, err bytes.Buffer
		if len(args) > 5 && args[5] != "" {
			args[5] = filepath.Join(t.TempDir(), "out")
		}
		if Run(args, &out, &err) != 1 || out.Len() != 0 || err.String() != help {
			t.Fatal("invalid args", args, err.String())
		}
		if len(args) > 5 && args[5] != "" {
			if _, e := os.Lstat(args[5]); !os.IsNotExist(e) {
				t.Fatal("invalid publication")
			}
		}
	}
	bad := filepath.Join(t.TempDir(), "bad.top")
	if e := os.WriteFile(bad, []byte("bad"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"missing.top", bad, "../exporttext/testdata/mixed.top"} {
		dir := filepath.Join(t.TempDir(), "out")
		var out, err bytes.Buffer
		if Run([]string{"export", path, "--format", "walls", "--output", dir}, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("input failure", path, err.String())
		}
		if _, e := os.Lstat(dir); !os.IsNotExist(e) {
			t.Fatal("failure published")
		}
	}
	dir := filepath.Join(t.TempDir(), "out")
	var err bytes.Buffer
	if Run([]string{"export", input, "--format", "walls", "--output", dir}, dxfFailingWriter{}, &err) != 1 || !strings.Contains(err.String(), "export published") {
		t.Fatal("notification failure", err.String())
	}
	if _, e := os.Stat(filepath.Join(dir, "report.json")); e != nil {
		t.Fatal("package lost", e)
	}
	b, e := os.ReadFile(filepath.Join(dir, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var p exportwalls.Report
	if e = json.Unmarshal(b, &p); e != nil || p.Format != "walls-srv" || len(p.Measurements) != 5 {
		t.Fatal("report", e)
	}
}
