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
	"pockettopo-exporter/internal/exporttoporobot"
	"strings"
	"testing"
	"time"
)

func TestToporobotCLIEmptyNative(t *testing.T) {
	target := filepath.Join(t.TempDir(), "out")
	var out, err bytes.Buffer
	status := cli.Run([]string{"export", "../processing/testdata/geometry/empty.top", "--format", "toporobot", "--output", target, "--export-time", "2026-10-13T12:34:56"}, &out, &err)
	if status != 0 {
		t.Fatalf("Toporobot missing: status=%d %s", status, &err)
	}
	b, e := os.ReadFile(filepath.Join(target, "survey.text"))
	if e != nil || string(b) != "    -1     1   1   1   1  360.00  360.00    0.05    1.00    1.00  100.00  100.00\r\n" {
		t.Fatalf("native empty bytes %q, %v", b, e)
	}
}

func TestToporobotCLIOptionsPublication(t *testing.T) {
	for _, tc := range []struct {
		golden, input string
		options       []string
	}{
		{"options-400-15", "../exportvisualtopo/testdata/dimensions.top", []string{"--angle-unit", "grads", "--code", "7", "--trip", "9", "--auto-declination", "--manual-declination", "--vertical-sections", "--include-splays", "true"}},
		{"options-360-0", "../exportvisualtopo/testdata/dimensions.top", []string{"--angle-unit", "degrees", "--code", "7", "--trip", "9", "--include-splays", "false"}},
		{"c360-closed-context-crossfile", "../processing/testdata/context/crossfile/current.top", []string{"--template-directory", "../processing/testdata/context/crossfile", "--close-loops"}},
		{"merge-15", "../exportvisualtopo/testdata/dimensions.top", []string{"--angle-unit", "grads", "--code", "7", "--trip", "9", "--auto-declination", "--manual-declination", "--vertical-sections", "--add-to", "../exporttoporobot/testdata/merge-base.text"}},
	} {
		t.Run(tc.golden, func(t *testing.T) {
			want, e := os.ReadFile("../exporttoporobot/testdata/" + tc.golden + ".text")
			if e != nil {
				t.Fatal(e)
			}
			at := "2026-10-13T12:34:56"
			for _, line := range strings.Split(string(want), "\r\n") {
				if strings.HasPrefix(line, "    -4") {
					v, e := time.Parse("06/01/02 15:04:05", line[25:42])
					if e != nil {
						t.Fatal(e)
					}
					at = v.Format("2006-01-02T15:04:05")
					break
				}
			}
			target := filepath.Join(t.TempDir(), "out")
			args := []string{"export", tc.input, "--format", "toporobot", "--output", target, "--export-time", at}
			args = append(args, tc.options...)
			var out, err bytes.Buffer
			if cli.Run(args, &out, &err) != 0 || err.Len() != 0 || out.String() != "Exported survey.text and report.json\n" {
				t.Fatal(out.String(), err.String())
			}
			got, e := os.ReadFile(filepath.Join(target, "survey.text"))
			if e != nil || !bytes.Equal(got, want) {
				t.Fatal("native CLI bytes", e)
			}
			jb, e := os.ReadFile(filepath.Join(target, "report.json"))
			if e != nil {
				t.Fatal(e)
			}
			var r exporttoporobot.Report
			if e = json.Unmarshal(jb, &r); e != nil {
				t.Fatal(e)
			}
			if r.OutputHash != fmt.Sprintf("%x", sha256.Sum256(want)) || r.OutputSize != len(want) || len(r.Inputs) < 1 || r.Inputs[len(r.Inputs)-1].Name == "" {
				t.Fatal("report inventory", string(jb))
			}
			dir, existing := "", ""
			for i, flag := range tc.options {
				if flag == "--template-directory" {
					dir = tc.options[i+1]
				}
				if flag == "--add-to" {
					existing = tc.options[i+1]
				}
			}
			context := "isolated-input"
			if dir != "" {
				context = "explicit-template-directory"
			}
			if r.Context != context {
				t.Fatal("context label", r.Context, context)
			}
			current, merges := 0, 0
			for _, input := range r.Inputs {
				path := filepath.Join(filepath.Dir(tc.input), input.Name)
				if input.Role == "current" {
					current++
					if input.Name != filepath.Base(tc.input) {
						t.Fatal("current name", input)
					}
				}
				if input.Role == "existing-toporobot" {
					merges++
					path = existing
					if input.Name != filepath.Base(existing) {
						t.Fatal("existing name", input)
					}
				}
				b, e := os.ReadFile(path)
				if e != nil || input.SHA256 != fmt.Sprintf("%x", sha256.Sum256(b)) || input.Size != len(b) || input.Consumed+input.Ignored != len(b) {
					t.Fatal("exact input inventory", input, e)
				}
			}
			if current != 1 || (existing != "" && merges != 1) || (existing == "" && merges != 0) {
				t.Fatal("input roles", r.Inputs)
			}
			if existing != "" {
				b, _ := os.ReadFile(existing)
				if r.ExistingHash != fmt.Sprintf("%x", sha256.Sum256(b)) || r.ExistingSize != len(b) {
					t.Fatal("existing report identity")
				}
			}
			if cli.Run(args, io.Discard, &err) != 1 {
				t.Fatal("existing destination accepted")
			}
			again, _ := os.ReadFile(filepath.Join(target, "survey.text"))
			if !bytes.Equal(got, again) {
				t.Fatal("output changed")
			}
		})
	}
}

func TestToporobotCLIInclusiveOptionsAndEmptyContext(t *testing.T) {
	for _, n := range []string{"1", "100000"} {
		var err bytes.Buffer
		args := []string{"export", "../processing/testdata/geometry/empty.top", "--format", "toporobot", "--output", filepath.Join(t.TempDir(), "out"), "--code", n, "--trip", n, "--export-time", "0001-01-01T00:00:00"}
		if cli.Run(args, io.Discard, &err) != 0 {
			t.Fatal("inclusive dialog bounds", n, err.String())
		}
	}
	dir := t.TempDir()
	b, e := os.ReadFile("../processing/testdata/geometry/empty.top")
	if e != nil {
		t.Fatal(e)
	}
	input := filepath.Join(dir, "current.top")
	if e = os.WriteFile(input, b, 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "out")
	if cli.Run([]string{"export", input, "--format", "toporobot", "--output", target, "--template-directory", dir}, io.Discard, io.Discard) != 0 {
		t.Fatal("empty explicit context")
	}
	jb, e := os.ReadFile(filepath.Join(target, "report.json"))
	var r exporttoporobot.Report
	if e != nil || json.Unmarshal(jb, &r) != nil || r.Context != "explicit-template-directory" || len(r.Inputs) != 1 {
		t.Fatal("explicit empty context report", string(jb), e)
	}
}
func TestToporobotCLIFailures(t *testing.T) {
	base := []string{"export", "../processing/testdata/geometry/empty.top", "--format", "toporobot", "--output", filepath.Join(t.TempDir(), "out")}
	for _, tail := range [][]string{{"--unknown"}, {"--angle-unit"}, {"--angle-unit", "radians"}, {"--code", "0"}, {"--code", "100001"}, {"--trip", "no"}, {"--include-splays", "no"}, {"--export-time", "no"}, {"--export-time", "0000-01-01T00:00:00"}, {"--close-loops", "--close-loops"}, {"--auto-declination", "--auto-declination"}, {"--add-to", ""}, {"--template-directory", "missing"}, {"--add-to", "missing"}} {
		var out, err bytes.Buffer
		if cli.Run(append(append([]string{}, base...), tail...), &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal(tail)
		}
	}
	for _, args := range [][]string{{"export", "a", "--format", "toporobot"}, {"export", "--json", "--format", "toporobot", "--output", "out"}, {"export", "a", "--format", "toporobot", "--no", "out"}, {"export", "a", "--format", "toporobot", "--output", ""}, {"export", "missing", "--format", "toporobot", "--output", "out"}, {"export", "../exportvisualtopo/testdata/extreme.top", "--format", "toporobot", "--output", "out"}} {
		if len(args) >= 6 {
			args[5] = filepath.Join(t.TempDir(), "out")
		}
		if cli.Run(args, io.Discard, io.Discard) != 1 {
			t.Fatal(args)
		}
	}
	bad := filepath.Join(t.TempDir(), "bad.top")
	os.WriteFile(bad, []byte("bad"), 0600)
	if cli.Run([]string{"export", bad, "--format", "toporobot", "--output", "out"}, io.Discard, io.Discard) != 1 {
		t.Fatal("malformed input")
	}
	args := append([]string{}, base...)
	args[5] = filepath.Join(t.TempDir(), "out")
	if cli.Run(args, failingWriter{}, io.Discard) != 1 {
		t.Fatal("notification failure")
	}
	if _, e := os.Stat(filepath.Join(args[5], "report.json")); e != nil {
		t.Fatal("published output removed", e)
	}
	invalid := filepath.Join(t.TempDir(), "bad.text")
	os.WriteFile(invalid, []byte("bad"), 0600)
	if cli.Run(append(base, "--add-to", invalid), io.Discard, io.Discard) != 1 {
		t.Fatal("malformed merge")
	}
}
