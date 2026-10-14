package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellExportPreservesSourceAndExistingExport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "spaces Żółć %1 %L bare% $ apostrophe's")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../internal/top/testdata/api-drawings.top")
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "cave Żółć %1 %L bare% $ apostrophe's.top")
	if err := os.WriteFile(input, source, 0600); err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSuffix(input, ".top") + "-text"
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(base, "keep.txt")
	if err := os.WriteFile(marker, []byte("foreign export"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{base + "-2", base + "-3"} {
		var stdout, stderr bytes.Buffer
		if code := runShellExport([]string{"shell-export", input, "--format", "text"}, &stdout, &stderr); code != 0 {
			t.Fatalf("export status %d: %s", code, &stderr)
		}
		for _, name := range []string{"survey.txt", "report.json"} {
			if _, err := os.Stat(filepath.Join(output, name)); err != nil {
				t.Fatal(err)
			}
		}
		if !strings.Contains(stdout.String(), output) || !strings.Contains(stdout.String(), filepath.Join(output, "report.json")) {
			t.Fatalf("missing output/report location: %s", &stdout)
		}
	}
	if got, err := os.ReadFile(input); err != nil || !bytes.Equal(got, source) {
		t.Fatal("source changed", err)
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "foreign export" {
		t.Fatal("existing export changed", err)
	}
}

func TestShellExportRejectsInvalidInputsWithoutPublication(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "bad.top")
	if err := os.WriteFile(input, []byte("not a TOP"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"shell-export", input, "--format", "text"},
		{"shell-export", input, "--format", "unknown"},
		{"shell-export", input, "--format", "text", "--view", "side"},
		{"shell-export", input, "--format", "dxf", "--view", "invalid"},
		{"shell-export", input, "--format", "dxf", "--output", "elsewhere"},
		{"shell-export", input},
		{"shell-export", filepath.Join(dir, "missing.top"), "--format", "text"},
	} {
		var stdout, stderr bytes.Buffer
		if code := runShellExport(args, &stdout, &stderr); code != 1 || stderr.Len() == 0 {
			t.Fatalf("missing readable rejection for %v: code %d, %s", args, code, &stderr)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("failure left output", entries, err)
	}
}

func TestShellExportBoundedCollision(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "cave.top")
	for n := 1; n <= 1000; n++ {
		name := strings.TrimSuffix(input, ".top") + "-text"
		if n > 1 {
			name += "-" + fmt.Sprint(n)
		}
		if err := os.Mkdir(name, 0700); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runShellExport([]string{"shell-export", input, "--format", "text"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "1000") {
		t.Fatalf("collision bound: %d %s", code, &stderr)
	}
	if _, err := os.Stat(strings.TrimSuffix(input, ".top") + "-text-1001"); !os.IsNotExist(err) {
		t.Fatal("crossed collision bound", err)
	}
}

func TestMenuCommandQuotesExecutableWithoutShellExpansion(t *testing.T) {
	executable := `C:\Program Files\Żółć $ apostrophe's & test\exporter.exe`
	want := `"C:\Program Files\Żółć $ apostrophe's & test\exporter.exe" shell-export "%1" --format dxf --view side`
	if got := menuCommand(executable, "dxf", "side"); got != want {
		t.Fatalf("direct command: %q", got)
	}
}

func TestMenuRejectsEveryExecutablePercentToken(t *testing.T) {
	for _, executable := range []string{
		`C:\portable %1\exporter.exe`,
		`C:\portable %L\exporter.exe`,
		`C:\portable %2\exporter.exe`,
		`C:\portable bare%\exporter.exe`,
		`C:\portable\exporter%.exe`,
	} {
		err := menuExecutablePathError(executable)
		if err == nil || !strings.Contains(err.Error(), "must not contain %") || !strings.Contains(err.Error(), "move the program folder and retry") {
			t.Fatalf("unsafe executable registration path accepted: %q, %v", executable, err)
		}
	}
	if err := menuExecutablePathError(`C:\spaces Żółć $ apostrophe's & test\exporter.exe`); err != nil {
		t.Fatal("safe literal executable path rejected", err)
	}
}

func TestShellExportAllMenuProfilesUseExistingExportDefaults(t *testing.T) {
	for _, tc := range []struct{ format, view, directory, artifact string }{
		{"text", "", "text", "survey.txt"},
		{"svx", "", "survex", "survey.svx"},
		{"srv", "", "walls", "survey.srv"},
		{"therion", "", "therion", "survey.txt"},
		{"visualtopo", "", "visualtopo", "survey.tro"},
		{"toporobot", "", "toporobot", "survey.text"},
		{"dxf", "plan", "dxf-plan", "survey.dxf"},
		{"dxf", "side", "dxf-side", "survey.dxf"},
		{"dxf3d", "", "dxf3d", "survey.dxf"},
	} {
		t.Run(tc.directory, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "cave.TOP")
			fixture := "../../internal/top/testdata/api-drawings.top"
			if tc.format == "toporobot" {
				fixture = "../../internal/exporttoporobot/testdata/topology.top"
			}
			source, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(input, source, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"shell-export", input, "--format", tc.format}
			if tc.view != "" {
				args = append(args, "--view", tc.view)
			}
			var stdout, stderr bytes.Buffer
			if code := runShellExport(args, &stdout, &stderr); code != 0 {
				t.Fatalf("%d %s", code, &stderr)
			}
			output := strings.TrimSuffix(input, ".TOP") + "-" + tc.directory
			if _, err := os.Stat(filepath.Join(output, tc.artifact)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(output, "report.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
