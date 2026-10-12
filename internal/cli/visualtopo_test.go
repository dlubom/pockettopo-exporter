package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/cli"
	"pockettopo-exporter/internal/exportvisualtopo"
	"testing"
)

func TestVisualTopoCLIEmptyNative(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer
	status := cli.Run([]string{"export", "../processing/testdata/geometry/empty.top", "--format", "visualtopo", "--output", out}, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("VisualTopo export status %d: %s", status, stderr.String())
	}
	b, e := os.ReadFile(filepath.Join(out, "survey.tro"))
	if e != nil {
		t.Fatal(e)
	}
	want := "Version 5.02\r\n"
	if string(b) != want {
		t.Fatalf("native empty bytes %q", b)
	}
}

func TestVisualTopoCLIOptionsAndPublication(t *testing.T) {
	input := "../exportvisualtopo/testdata/features.top"
	var out, err bytes.Buffer
	target := filepath.Join(t.TempDir(), "out")
	args := []string{"export", input, "--format", "visualtopo", "--output", target, "--close-loops"}
	if cli.Run(args, &out, &err) != 0 || err.Len() != 0 || out.String() != "Exported survey.tro and report.json\n" {
		t.Fatalf("export %s %s", &out, &err)
	}
	want, e := os.ReadFile("../exportvisualtopo/testdata/circle360/features-closed.txt")
	if e != nil {
		t.Fatal(e)
	}
	got, e := os.ReadFile(filepath.Join(target, "survey.tro"))
	if e != nil || !bytes.Equal(got, want) {
		t.Fatal("CLI golden", e)
	}
	raw, e := os.ReadFile(input)
	if e != nil {
		t.Fatal(e)
	}
	jb, e := os.ReadFile(filepath.Join(target, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var report exportvisualtopo.Report
	if e = json.Unmarshal(jb, &report); e != nil {
		t.Fatal(e)
	}
	if report.Context != "isolated-input" || !report.Closed || report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(raw)) || report.OutputHash != fmt.Sprintf("%x", sha256.Sum256(want)) || len(report.Inputs) != 1 {
		t.Fatal("isolated report", string(jb))
	}
	si := report.Inputs[0]
	if si.Name != "features.top" || si.Role != "current" || si.Size != len(raw) || si.Consumed != len(raw) || si.Ignored != 0 || si.SHA256 != report.SourceHash {
		t.Fatal("isolated inventory", si)
	}
	before := append([]byte{}, got...)
	out.Reset()
	err.Reset()
	if cli.Run(args, &out, &err) != 1 || out.Len() != 0 {
		t.Fatal("existing destination accepted")
	}
	got, _ = os.ReadFile(filepath.Join(target, "survey.tro"))
	if !bytes.Equal(got, before) {
		t.Fatal("existing export changed")
	}
	entries, e := os.ReadDir(target)
	if e != nil || len(entries) != 2 {
		t.Fatal("publication artifacts", entries, e)
	}
	for _, tail := range [][]string{{"--close-loops", "--close-loops"}, {"--template-directory"}, {"--template-directory", ""}, {"--template-directory", ".", "--template-directory", "."}, {"--unknown"}} {
		a := append([]string{"export", input, "--format", "visualtopo", "--output", filepath.Join(t.TempDir(), "out")}, tail...)
		out.Reset()
		err.Reset()
		if cli.Run(a, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("invalid options", tail)
		}
	}
	for _, a := range [][]string{{"export", input, "--format", "visualtopo"}, {"export", "", "--format", "visualtopo", "--output", "x"}, {"export", input, "--format", "visualtopo", "--bad", "x"}, {"export", input, "--format", "visualtopo", "--output", ""}, {"export", "--format", "--format", "visualtopo", "--output", "x"}, {"export", "missing.top", "--format", "visualtopo", "--output", "x"}, {"export", input, "--format", "visualtopo", "--output", filepath.Join(t.TempDir(), "missing", "out")}} {
		out.Reset()
		err.Reset()
		if cli.Run(a, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("invalid export", a)
		}
	}
}

func TestVisualTopoCLIContext(t *testing.T) {
	dir := "../exportvisualtopo/testdata/template-rays"
	target := filepath.Join(t.TempDir(), "out")
	var out, err bytes.Buffer
	args := []string{"export", dir + "/current.top", "--format", "visualtopo", "--output", target, "--template-directory", dir, "--close-loops"}
	if cli.Run(args, &out, &err) != 0 {
		t.Fatal("context export", &err)
	}
	want, e := os.ReadFile("../exportvisualtopo/testdata/circle360/nativecontext-template-rays-closed.txt")
	if e != nil {
		t.Fatal(e)
	}
	got, e := os.ReadFile(filepath.Join(target, "survey.tro"))
	if e != nil || !bytes.Equal(got, want) {
		t.Fatal("context native bytes", e)
	}
	report, e := os.ReadFile(filepath.Join(target, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var r struct {
		Schema  int    `json:"schema_version"`
		Context string `json:"context"`
		Closed  bool   `json:"close_loops"`
		Inputs  []exportvisualtopo.InputInfo
	}
	if e = json.Unmarshal(report, &r); e != nil {
		t.Fatal(e)
	}
	if r.Schema != 1 || r.Context != "explicit-template-directory" || !r.Closed || len(r.Inputs) != 2 || r.Inputs[0].Name != "a.top" || r.Inputs[0].Role != "template" || len(r.Inputs[0].SHA256) != 64 || r.Inputs[1].Name != "current.top" {
		t.Fatal("context report", string(report))
	}
	bad := append([]string{}, args...)
	bad[7] = "missing-directory"
	if cli.Run(bad, io.Discard, io.Discard) != 1 {
		t.Fatal("bad context accepted")
	}
	if cli.Run([]string{"export", dir + "/current.top", "--format", "visualtopo", "--output", filepath.Join(t.TempDir(), "out")}, failingWriter{}, io.Discard) != 1 {
		t.Fatal("notification failure ignored")
	}
}

func TestVisualTopoCLIGrads(t *testing.T) {
	input := "../exportvisualtopo/testdata/dimensions.top"
	out := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer
	if cli.Run([]string{"export", input, "--format", "visualtopo", "--output", out, "--angle-unit", "grads"}, &stdout, &stderr) != 0 {
		t.Fatal(stderr.String())
	}
	b, e := os.ReadFile(filepath.Join(out, "survey.tro"))
	want, e2 := os.ReadFile("../exportvisualtopo/testdata/circle400/dimensions.txt")
	if e != nil || e2 != nil || !bytes.Equal(b, want) {
		t.Fatal("native grads", e, e2)
	}
	for _, tail := range [][]string{{"--angle-unit"}, {"--angle-unit", ""}, {"--angle-unit", "radians"}, {"--angle-unit", "degrees", "--angle-unit", "grads"}} {
		stdout.Reset()
		stderr.Reset()
		args := append([]string{"export", input, "--format", "visualtopo", "--output", filepath.Join(t.TempDir(), "out")}, tail...)
		if cli.Run(args, &stdout, &stderr) != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("bad angle option", tail)
		}
	}
}
