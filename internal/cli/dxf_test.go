package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exportdxf"
	"strings"
	"testing"
)

func TestDXFExport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "export")
	var stdout, stderr bytes.Buffer
	args := []string{"export", "../top/testdata/api-drawings.top", "--format", "dxf", "--output", out}
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("DXF export: %d %s", code, stderr.String())
	}
	b, e := os.ReadFile(filepath.Join(out, "survey.dxf"))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.HasPrefix(b, []byte("0\r\nSECTION\r\n")) || !bytes.HasSuffix(b, []byte("0\r\nEOF\r\n")) {
		t.Fatal("complete native DXF framing")
	}
	if _, e = os.Stat(filepath.Join(out, "report.json")); e != nil {
		t.Fatal(e)
	}
	if stdout.String() != "Exported survey.dxf and report.json\n" {
		t.Fatal(stdout.String())
	}
	if Run(args, &stdout, &stderr) != 1 {
		t.Fatal("replaced output")
	}
}

func TestDXFCLIOptionsAndFailures(t *testing.T) {
	input := "../exportdxf/testdata/features.top"
	base := []string{"export", input, "--format", "dxf", "--output", "unused"}
	bad := [][]string{{"export", input, "--format", "dxf"}, {"export", "", "--format", "dxf", "--output", "x"}, {"export", "--format", "--format", "dxf", "--output", "x"}, {"export", input, "--format", "dxf", "bad", "x"}, {"export", input, "--format", "dxf", "--output", ""}}
	for _, extra := range [][]string{{"--scale"}, {"--scale", ""}, {"--scale", "0"}, {"--scale", "100001"}, {"--scale", "bad"}, {"--scale", "3", "--scale", "4"}, {"--view", "bad"}, {"--dxf-options", "1"}, {"--dxf-options", "4096"}, {"--dxf-options", "bad"}, {"--grid-unit", "yards"}, {"--unknown", "value"}, {"--template-directory", "a", "--template-directory", "b"}, {"--close-loops", "--close-loops"}} {
		bad = append(bad, append(append([]string{}, base...), extra...))
	}
	for _, args := range bad {
		var out, err bytes.Buffer
		if Run(args, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal("bad DXF args", args, &out, &err)
		}
	}
	var stdout, stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "out")
	args := []string{"export", input, "--format", "dxf", "--output", target, "--view", "side", "--scale", "3", "--dxf-options", "3582", "--grid-unit", "metres", "--close-loops"}
	if Run(args, &stdout, &stderr) != 0 {
		t.Fatal(stderr.String())
	}
	b, e := os.ReadFile(filepath.Join(target, "survey.dxf"))
	if e != nil {
		t.Fatal(e)
	}
	// Scale3/side native capture is open; closed default scale500 has separate bytes.
	raw, e := os.ReadFile(input)
	if e != nil {
		t.Fatal(e)
	}
	jb, e := os.ReadFile(filepath.Join(target, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	var r exportdxf.Report
	if e = json.Unmarshal(jb, &r); e != nil {
		t.Fatal(e)
	}
	if r.Drawing != "side" || r.Options.Mask != 3583 || r.Options.Scale != 3 || r.Options.Feet || !r.Options.Geometry.CloseLoops || r.Context != "isolated-input" || r.OutputHash != fmt.Sprintf("%x", sha256.Sum256(b)) || r.SourceHash != fmt.Sprintf("%x", sha256.Sum256(raw)) || len(r.Inputs) != 1 {
		t.Fatal("DXF report", r)
	}
	inv := r.Inputs[0]
	if inv.Name != "features.top" || inv.Role != "current" || inv.SHA256 != r.SourceHash || inv.Size != len(raw) || inv.Ignored != 0 || inv.Consumed != len(raw) {
		t.Fatal("DXF inventory", inv)
	}
	// Failures publish nothing, including parse, native partial and input I/O failures.
	for _, path := range []string{"missing.top", "../exportdxf/testdata/zero-polygons.top"} {
		target = filepath.Join(t.TempDir(), "out")
		stdout.Reset()
		stderr.Reset()
		if Run([]string{"export", path, "--format", "dxf", "--output", target}, &stdout, &stderr) != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("DXF failure", path)
		}
		if _, e = os.Lstat(target); !os.IsNotExist(e) {
			t.Fatal("failure published", e)
		}
	}
	malformed := filepath.Join(t.TempDir(), "bad.top")
	if e = os.WriteFile(malformed, []byte("bad"), 0600); e != nil {
		t.Fatal(e)
	}
	if Run([]string{"export", malformed, "--format", "dxf", "--output", filepath.Join(t.TempDir(), "out")}, &stdout, &stderr) != 1 {
		t.Fatal("malformed")
	}
	// Notification failure preserves the complete package.
	target = filepath.Join(t.TempDir(), "out")
	stderr.Reset()
	if Run([]string{"export", input, "--format", "dxf", "--output", target}, dxfFailingWriter{}, &stderr) != 1 || !strings.Contains(stderr.String(), "export published") {
		t.Fatal("DXF notification", stderr.String())
	}
	if _, e = os.Stat(filepath.Join(target, "report.json")); e != nil {
		t.Fatal(e)
	}
}
func TestDXFCLIContextAndFeetGolden(t *testing.T) {
	for _, feet := range []bool{false, true} {
		input := "../exportdxf/testdata/features.top"
		extra := []string{"--dxf-options", "3582"}
		golden := "../exportdxf/testdata/features-feet-0.dxf"
		if feet {
			extra = append(extra, "--grid-unit", "feet")
		} else {
			input = "../exportdxf/testdata/template-rays/current.top"
			extra = []string{"--template-directory", filepath.Dir(input), "--dxf-options", "510", "--close-loops"}
			golden = "../exportdxf/testdata/nativecontext-template-rays-0-closed.dxf"
		}
		target := filepath.Join(t.TempDir(), "out")
		args := append([]string{"export", input, "--format", "dxf", "--output", target}, extra...)
		var out, err bytes.Buffer
		if Run(args, &out, &err) != 0 {
			t.Fatal(err.String())
		}
		got, e := os.ReadFile(filepath.Join(target, "survey.dxf"))
		if e != nil {
			t.Fatal(e)
		}
		want, e := os.ReadFile(golden)
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("DXF CLI native bytes")
		}
		jb, e := os.ReadFile(filepath.Join(target, "report.json"))
		if e != nil {
			t.Fatal(e)
		}
		var r exportdxf.Report
		if e = json.Unmarshal(jb, &r); e != nil {
			t.Fatal(e)
		}
		if feet {
			if !r.Options.Feet {
				t.Fatal("feet")
			}
		} else {
			if r.Context != "explicit-template-directory" || len(r.Inputs) != 2 || r.Inputs[0].Name != "a.top" || r.Inputs[0].Role != "template" || r.Inputs[0].Ignored == 0 || r.Inputs[1].Role != "current" {
				t.Fatal("context report", r)
			}
		}
	}
	var out, err bytes.Buffer
	if Run([]string{"export", "missing.top", "--format", "dxf", "--output", "unused", "--template-directory", "missing"}, &out, &err) != 1 {
		t.Fatal("context failure")
	}
	target := filepath.Join(t.TempDir(), "existing")
	os.Mkdir(target, 0700)
	if Run([]string{"export", "../exportdxf/testdata/features.top", "--format", "dxf", "--output", target}, &out, &err) != 1 {
		t.Fatal("DXF publication failure")
	}
}

type dxfFailingWriter struct{}

func (dxfFailingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("output failed") }
