package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exportsurvex"
	"strings"
	"testing"
)

func TestSurvexCLIExport(t *testing.T) {
	var out, err bytes.Buffer
	dir := filepath.Join(t.TempDir(), "out")
	args := []string{"export", "../exportsurvex/testdata/features.top", "--format", "survex", "--output", dir}
	if Run(args, &out, &err) != 0 {
		t.Fatal("Survex export:", err.String())
	}
	got, e := os.ReadFile(filepath.Join(dir, "survey.svx"))
	if e != nil {
		t.Fatal(e)
	}
	want, e := os.ReadFile("../exportsurvex/testdata/features.svx")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Survex golden: %s", got)
	}
	if out.String() != "Exported survey.svx and report.json\n" {
		t.Fatal("notification", out.String())
	}
	if Run(args, &out, &err) != 1 {
		t.Fatal("replaced existing output")
	}
}

func TestSurvexCLIOptionsAndFailures(t *testing.T) {
	input := "../exportsurvex/testdata/features.top"
	for _, args := range [][]string{
		{"export", input, "--format", "survex"}, {"export", "", "--format", "survex", "--output", "x"},
		{"export", "--bad", "--format", "survex", "--output", "x"}, {"export", input, "--format", "survex", "bad", "x"},
		{"export", input, "--format", "survex", "--output", ""}, {"export", input, "--format", "survex", "--output", "x", "--close-loops"},
		{"export", input, "--format", "survex", "--output", "x", "--template-directory", "."},
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
		if Run([]string{"export", path, "--format", "survex", "--output", dir}, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("input failure", path, err.String())
		}
		if _, e := os.Lstat(dir); !os.IsNotExist(e) {
			t.Fatal("failure published")
		}
	}
	dir := filepath.Join(t.TempDir(), "out")
	var err bytes.Buffer
	if Run([]string{"export", input, "--format", "survex", "--output", dir}, dxfFailingWriter{}, &err) != 1 || !strings.Contains(err.String(), "export published") {
		t.Fatal("notification failure", err.String())
	}
	if _, e := os.Stat(filepath.Join(dir, "report.json")); e != nil {
		t.Fatal("package lost", e)
	}
	b, e := os.ReadFile(filepath.Join(dir, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var p exportsurvex.Report
	if e = json.Unmarshal(b, &p); e != nil || p.Format != "survex-svx" || len(p.Measurements) != 5 {
		t.Fatal("report", e)
	}
}
