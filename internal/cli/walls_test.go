package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWallsExport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "export")
	args := []string{"export", "../processing/testdata/repeats.top", "--format", "srv", "--output", dir}
	var out, err bytes.Buffer
	if Run(args, &out, &err) != 0 {
		t.Fatal(err.String())
	}
	b, e := os.ReadFile(filepath.Join(dir, "survey.srv"))
	if e != nil || !bytes.Contains(b, []byte("#Units Decl=")) || bytes.Contains(b, []byte("Average of")) {
		t.Fatal("SRV", e, string(b))
	}
	if out.String() != "Exported survey.srv and report.json\n" {
		t.Fatal("notification", out.String())
	}
	if _, e = os.ReadFile(filepath.Join(dir, "report.json")); e != nil {
		t.Fatal(e)
	}
	if Run(args, &out, &err) != 1 {
		t.Fatal("overwritten")
	}
	if Run([]string{"export", args[1], "--format", "srv", "--output", filepath.Join(t.TempDir(), "out")}, dxfFailingWriter{}, &err) != 1 || !strings.Contains(err.String(), "export published") {
		t.Fatal("notification failure", err.String())
	}
}

func TestWallsCLIInvalid(t *testing.T) {
	for _, args := range [][]string{
		{"export", "x", "--format", "srv"},
		{"export", "", "--format", "srv", "--output", "x"},
		{"export", "--bad", "--format", "srv", "--output", "x"},
		{"export", "x", "--format", "srv", "wrong", "x"},
		{"export", "x", "--format", "srv", "--output", ""},
		{"export", "x", "--format", "srv", "--output", "x", "--close-loops"},
		{"export", "x", "--format", "srv", "--output", "x", "--template-directory", "."},
	} {
		var out, err bytes.Buffer
		if Run(args, &out, &err) != 1 || out.Len() != 0 || err.String() != help {
			t.Fatal(args, err.String())
		}
	}
	for _, input := range []string{"missing.top", "../top/testdata/plan-third-next-marker-unsupported.top", "../processing/testdata/boundaries.top"} {
		dir := filepath.Join(t.TempDir(), "out")
		var out, err bytes.Buffer
		if Run([]string{"export", input, "--format", "srv", "--output", dir}, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal(input, err.String())
		}
		if _, e := os.Lstat(dir); !os.IsNotExist(e) {
			t.Fatal("published failure", e)
		}
	}
}
