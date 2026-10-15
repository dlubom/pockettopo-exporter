package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pockettopo-exporter/internal/cli"
)

func TestBuildVersionInVersionAndInspection(t *testing.T) {
	previous := cli.Version
	t.Cleanup(func() { cli.Version = previous })
	cli.Version = "1.0.0"
	var out, diagnostic bytes.Buffer
	if status := cli.Run([]string{"--version"}, &out, &diagnostic); status != 0 || out.String() != "pockettopo-exporter 1.0.0\n" || diagnostic.Len() != 0 {
		t.Fatalf("version: status=%d stdout=%q stderr=%q", status, out.String(), diagnostic.String())
	}
	out.Reset()
	if status := cli.Run([]string{"inspect", "../top/testdata/api-drawings.top", "--json"}, &out, &diagnostic); status != 0 || diagnostic.Len() != 0 {
		t.Fatalf("inspect: status=%d stderr=%q", status, diagnostic.String())
	}
	var result struct {
		Schema int    `json:"schema_version"`
		Tool   string `json:"tool_version"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Schema != 1 || result.Tool != "1.0.0" {
		t.Fatalf("inspection version: %+v error=%v", result, err)
	}
}

func TestBuildVersionInExportReports(t *testing.T) {
	previous := cli.Version
	t.Cleanup(func() { cli.Version = previous })
	cases := []struct {
		name, format, input, golden, artifact string
		options                               []string
	}{
		{"text", "text", "../exporttext/testdata/mixed.top", "../exporttext/testdata/mixed.txt", "survey.txt", nil},
		{"therion", "therion", "../exporttherion/testdata/features.top", "../exporttherion/testdata/features.txt", "survey.txt", nil},
		{"dxf-plan", "dxf", "../exportdxf/testdata/features.top", "../exportdxf/testdata/features-o0.dxf", "survey.dxf", []string{"--dxf-options", "0", "--view", "plan"}},
		{"dxf-side", "dxf", "../exportdxf/testdata/features.top", "../exportdxf/testdata/features-o1.dxf", "survey.dxf", []string{"--dxf-options", "0", "--view", "side"}},
		{"visualtopo", "visualtopo", "../exportvisualtopo/testdata/dimensions.top", "../exportvisualtopo/testdata/circle360/dimensions.txt", "survey.tro", nil},
		{"toporobot", "toporobot", "../exportvisualtopo/testdata/dimensions.top", "../exporttoporobot/testdata/c360-dimensions.text", "survey.text", []string{"--export-time", "2026-10-13T07:29:26"}},
		{"dxf3d", "dxf3d", "../exportdxf3d/testdata/fallback.top", "../exportdxf3d/testdata/fallback.dxf", "survey.dxf", nil},
		{"srv", "srv", "../exportwalls/testdata/features.top", "../exportwalls/testdata/features.srv", "survey.srv", nil},
		{"svx", "svx", "../exportwalls/testdata/features.top", "../exportsvx/testdata/lexical/features.svx", "survey.svx", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			golden, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatal(err)
			}
			var baseline []byte
			for _, version := range []string{"dev", "1.0.0", "2.3.4-rc.1+issue2"} {
				t.Run(version, func(t *testing.T) {
					cli.Version = version
					output := filepath.Join(t.TempDir(), "export")
					args := []string{"export", tc.input, "--format", tc.format, "--output", output}
					args = append(args, tc.options...)
					var out, diagnostic bytes.Buffer
					if code := cli.Run(args, &out, &diagnostic); code != 0 || diagnostic.Len() != 0 {
						t.Fatalf("export: status=%d stderr=%q", code, diagnostic.String())
					}
					artifact, err := os.ReadFile(filepath.Join(output, tc.artifact))
					if err != nil || !bytes.Equal(artifact, golden) {
						t.Fatalf("export bytes differ from independent golden: %v", err)
					}
					raw, err := os.ReadFile(filepath.Join(output, "report.json"))
					if err != nil {
						t.Fatal(err)
					}
					var report map[string]json.RawMessage
					if err := json.Unmarshal(raw, &report); err != nil {
						t.Fatal(err)
					}
					var got string
					if err := json.Unmarshal(report["tool_version"], &got); err != nil || got != version {
						t.Fatalf("report tool_version=%q, want %q; error=%v", got, version, err)
					}
					delete(report, "tool_version")
					normalized, err := json.Marshal(report)
					if err != nil {
						t.Fatal(err)
					}
					if version == "dev" {
						baseline = normalized
					} else if !bytes.Equal(normalized, baseline) {
						t.Fatal("build version changed report fields other than tool_version")
					}
				})
			}
		})
	}
}
