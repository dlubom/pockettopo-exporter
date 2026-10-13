package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exportdxf3d"
	"strings"
	"testing"
)

func TestDXF3DExport(t *testing.T) {
	var out, err bytes.Buffer
	dir := filepath.Join(t.TempDir(), "out")
	args := []string{"export", "../processing/testdata/geometry/chain.top", "--format", "dxf3d", "--output", dir}
	if Run(args, &out, &err) != 0 {
		t.Fatal("3D export", err.String())
	}
	got, e := os.ReadFile(filepath.Join(dir, "survey.dxf"))
	if e != nil {
		t.Fatal(e)
	}
	want, e := os.ReadFile("../exportdxf3d/testdata/geometry-chain.dxf")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("3D golden")
	}
	if out.String() != "Exported survey.dxf and report.json\n" {
		t.Fatal("notification")
	}
	if Run(args, &out, &err) != 1 {
		t.Fatal("replacement")
	}
}

func TestDXF3DCLIOptionsAndFailures(t *testing.T) {
	input := "../exportdxf3d/testdata/fallback.top"
	base := []string{"export", input, "--format", "dxf3d", "--output", "unused"}
	bad := [][]string{{"export", input, "--format", "dxf3d"}, {"export", "", "--format", "dxf3d", "--output", "x"}, {"export", "--bad", "--format", "dxf3d", "--output", "x"}, {"export", input, "--format", "dxf3d", "bad", "x"}, {"export", input, "--format", "dxf3d", "--output", ""}}
	for _, extra := range [][]string{{"--close-loops", "--close-loops"}, {"--template-directory"}, {"--template-directory", ""}, {"--template-directory", "--close-loops"}, {"--template-directory", "a", "--template-directory", "b"}, {"--view", "side"}, {"--scale", "500"}, {"--dxf-options", "14"}, {"--unknown"}} {
		bad = append(bad, append(append([]string{}, base...), extra...))
	}
	for _, args := range bad {
		var out, err bytes.Buffer
		if len(args) > 5 && args[5] != "" {
			args[5] = filepath.Join(t.TempDir(), "out")
		}
		if Run(args, &out, &err) != 1 || out.Len() != 0 || err.String() != help {
			t.Fatal("bad 3D args", args, err.String())
		}
		if len(args) > 5 && args[5] != "" {
			if _, e := os.Lstat(args[5]); !os.IsNotExist(e) {
				t.Fatal("invalid args published", e)
			}
		}
	}
	for _, path := range []string{"missing.top", "../processing/testdata/geometry/failure.top"} {
		var out, err bytes.Buffer
		dir := filepath.Join(t.TempDir(), "out")
		if Run([]string{"export", path, "--format", "dxf3d", "--output", dir}, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("3D failure", path)
		}
		if _, e := os.Lstat(dir); !os.IsNotExist(e) {
			t.Fatal("partial publication")
		}
	}
	path := filepath.Join(t.TempDir(), "bad.top")
	if e := os.WriteFile(path, []byte("bad"), 0600); e != nil {
		t.Fatal(e)
	}
	var out, err bytes.Buffer
	if Run([]string{"export", path, "--format", "dxf3d", "--output", filepath.Join(t.TempDir(), "out")}, &out, &err) != 1 {
		t.Fatal("parse error")
	}
	if Run(append(append([]string{}, base...), "--template-directory", "missing"), &out, &err) != 1 {
		t.Fatal("context error")
	}
	dir := filepath.Join(t.TempDir(), "out")
	if Run([]string{"export", input, "--format", "dxf3d", "--output", dir}, dxfFailingWriter{}, &err) != 1 || !strings.Contains(err.String(), "export published") {
		t.Fatal("notification failure", err.String())
	}
	if _, e := os.Stat(filepath.Join(dir, "report.json")); e != nil {
		t.Fatal("complete package lost", e)
	}
}
func TestDXF3DCLIContextAndReport(t *testing.T) {
	for _, mode := range []string{"isolated", "context", "empty-context"} {
		input := "../exportdxf3d/testdata/fallback.top"
		extra := []string{"--close-loops"}
		golden := "../exportdxf3d/testdata/fallback-closed.dxf"
		if mode == "context" {
			input = "../exportdxf/testdata/template-rays/current.top"
			extra = append(extra, "--template-directory", filepath.Dir(input))
			golden = "../exportdxf3d/testdata/template-rays-closed.dxf"
		}
		if mode == "empty-context" {
			extra = append(extra, "--template-directory", t.TempDir())
		}
		dir := filepath.Join(t.TempDir(), "out")
		var out, err bytes.Buffer
		args := append([]string{"export", input, "--format", "dxf3d", "--output", dir}, extra...)
		if Run(args, &out, &err) != 0 {
			t.Fatal(err.String())
		}
		got, e := os.ReadFile(filepath.Join(dir, "survey.dxf"))
		if e != nil {
			t.Fatal(e)
		}
		want, e := os.ReadFile(golden)
		if e != nil || !bytes.Equal(got, want) {
			t.Fatal("context native bytes", e)
		}
		b, e := os.ReadFile(filepath.Join(dir, "report.json"))
		if e != nil {
			t.Fatal(e)
		}
		var r exportdxf3d.Report
		if e = json.Unmarshal(b, &r); e != nil {
			t.Fatal(e)
		}
		context := "isolated-input"
		if mode != "isolated" {
			context = "explicit-template-directory"
		}
		if r.Context != context || !r.Options.Geometry.CloseLoops || r.OutputHash != fmt.Sprintf("%x", sha256.Sum256(got)) {
			t.Fatal("3D report", r)
		}
		inputs := []string{input}
		if mode == "context" {
			inputs = []string{filepath.Join(filepath.Dir(input), "a.top"), input}
		}
		if len(r.Inputs) != len(inputs) {
			t.Fatal("inventory cardinality")
		}
		for i, p := range inputs {
			raw, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			v := r.Inputs[i]
			role := "current"
			if i < len(inputs)-1 {
				role = "template"
			}
			if v.Name != filepath.Base(p) || v.Role != role || v.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) || v.Size != len(raw) || v.Consumed+v.Ignored != v.Size {
				t.Fatal("input inventory", v)
			}
			if role == "template" && v.Ignored == 0 {
				t.Fatal("template ignored drawing")
			}
		}
	}
}
