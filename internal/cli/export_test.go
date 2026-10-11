package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pockettopo-exporter/internal/cli"
)

func TestExportNativeEmpty(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "empty.top")
	// Independent literal complete empty v3 document, including writer trailer.
	b := []byte{'T', 'o', 'p', 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 244, 1, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 244, 1, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 244, 1, 0, 0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(input, b, 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "export")
	var out, diagnostic bytes.Buffer
	args := []string{"export", input, "--format", "text", "--output", output}
	if status := cli.Run(args, &out, &diagnostic); status != 0 || diagnostic.Len() != 0 || out.String() != "Exported survey.txt and report.json\n" {
		t.Fatalf("export: %d %q %q", status, out.String(), diagnostic.String())
	}
	text, err := os.ReadFile(filepath.Join(output, "survey.txt"))
	if err != nil || string(text) != "empty   (m, 360)\r\n\r\n\r\n\r\n" {
		t.Fatalf("text: %q, %v", text, err)
	}
	report, err := os.ReadFile(filepath.Join(output, "report.json"))
	if err != nil || !json.Valid(report) {
		t.Fatalf("report: %q, %v", report, err)
	}
	before, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(before, b) {
		t.Fatalf("input changed: %v", err)
	}
	// Re-running must fail without changing the first complete publication.
	out.Reset()
	diagnostic.Reset()
	if status := cli.Run(args, &out, &diagnostic); status != 1 || out.Len() != 0 || diagnostic.Len() == 0 {
		t.Fatalf("overwrite: %d %q %q", status, out.String(), diagnostic.String())
	}
	again, err := os.ReadFile(filepath.Join(output, "survey.txt"))
	if err != nil || !bytes.Equal(again, text) {
		t.Fatal("existing export changed")
	}
}

func TestExportInputAndDestinationFailures(t *testing.T) {
	dir := t.TempDir()
	valid, err := os.ReadFile("../exporttext/testdata/empty.top")
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "input.top")
	if err = os.WriteFile(input, valid, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		input string
		data  []byte
		dest  string
	}{
		{"missing", filepath.Join(dir, "missing.top"), nil, filepath.Join(dir, "missing-export")},
		{"directory", dir, nil, filepath.Join(dir, "directory-export")},
		{"malformed", filepath.Join(dir, "bad.top"), []byte("Top"), filepath.Join(dir, "bad-export")},
		{"unsupported version", filepath.Join(dir, "v4.top"), []byte("Top\x04"), filepath.Join(dir, "v4-export")},
		{"unsupported tail", filepath.Join(dir, "tail.top"), append(bytes.Clone(valid), 1), filepath.Join(dir, "tail-export")},
		{"control title", filepath.Join(dir, "bad\tname.top"), valid, filepath.Join(dir, "title-export")},
		{"missing parent", input, nil, filepath.Join(dir, "absent", "export")},
		{"input overwrite", input, nil, input},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.data != nil {
				if err := os.WriteFile(tc.input, tc.data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out, diagnostic bytes.Buffer
			if status := cli.Run([]string{"export", tc.input, "--format", "text", "--output", tc.dest}, &out, &diagnostic); status != 1 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "Cannot export:") {
				t.Fatalf("failure: %d %q %q", status, out.String(), diagnostic.String())
			}
			if tc.dest != input {
				if _, err := os.Lstat(tc.dest); !os.IsNotExist(err) {
					t.Fatalf("partial export: %v", err)
				}
			}
		})
	}
	before, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(before, valid) {
		t.Fatal("source overwritten")
	}
}

func TestExportArguments(t *testing.T) {
	for _, args := range [][]string{
		{"export"}, {"export", "", "--format", "text", "--output", "out"},
		{"export", "input", "--format", "dxf", "--output", "out"},
		{"export", "input", "--output", "text", "--format", "out"},
		{"export", "input", "--format", "text", "--output", ""},
		{"export", "input", "--format", "text", "--output", "out", "extra"},
	} {
		var out, diagnostic bytes.Buffer
		if status := cli.Run(args, &out, &diagnostic); status != 1 || out.Len() != 0 || diagnostic.Len() == 0 {
			t.Fatalf("bad arguments accepted: %v", args)
		}
	}
}

func TestExportNotificationFailureKeepsCompleteFiles(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output")
	var diagnostic bytes.Buffer
	if status := cli.Run([]string{"export", "../exporttext/testdata/mixed.top", "--format", "text", "--output", output}, failingWriter{}, &diagnostic); status != 1 || !strings.Contains(diagnostic.String(), "export published") {
		t.Fatalf("notification failure: %d %q", status, diagnostic.String())
	}
	for _, name := range []string{"survey.txt", "report.json"} {
		if b, err := os.ReadFile(filepath.Join(output, name)); err != nil || len(b) == 0 {
			t.Fatalf("published %s lost: %v", name, err)
		}
	}
}

func TestExportNativeCLIGoldensAndIsolation(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "mixed.top")
	b, err := os.ReadFile("../exporttext/testdata/mixed.top")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(input, b, 0600); err != nil {
		t.Fatal(err)
	}
	// A malicious neighbor must never be opened as native template context.
	if err = os.WriteFile(filepath.Join(dir, "neighbor.top"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	var reports [][]byte
	for _, name := range []string{"one", "two"} {
		output := filepath.Join(dir, name)
		var out, diagnostic bytes.Buffer
		if status := cli.Run([]string{"export", input, "--format", "text", "--output", output}, &out, &diagnostic); status != 0 || diagnostic.Len() != 0 {
			t.Fatalf("export: %d %q", status, diagnostic.String())
		}
		want, err := os.ReadFile("../exporttext/testdata/mixed.txt")
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(output, "survey.txt"))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("CLI golden mismatch: %v", err)
		}
		r, err := os.ReadFile(filepath.Join(output, "report.json"))
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, r)
		entries, err := os.ReadDir(output)
		if err != nil || len(entries) != 2 {
			t.Fatalf("publication debris: %v %v", entries, err)
		}
	}
	if !bytes.Equal(reports[0], reports[1]) {
		t.Fatal("output paths leaked into report")
	}
}

func TestExportInputReadFailure(t *testing.T) {
	input := t.TempDir()
	f, err := os.Open(input)
	if err != nil {
		t.Skip("platform rejects directory input on open")
	}
	_, readErr := io.ReadAll(f)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if readErr == nil {
		t.Skip("platform permits reading directory input")
	}
	output := filepath.Join(t.TempDir(), "output")
	var out, diagnostic bytes.Buffer
	if status := cli.Run([]string{"export", input, "--format", "text", "--output", output}, &out, &diagnostic); status != 1 || out.Len() != 0 || !strings.Contains(diagnostic.String(), readErr.Error()) {
		t.Fatalf("input I/O error lost: %d %q %q, want %v", status, out.String(), diagnostic.String(), readErr)
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("failed input published: %v", err)
	}
}

func TestExportInputBudget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "large.top")
	output := filepath.Join(dir, "output")
	f, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../exporttext/testdata/empty.top")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(b); err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(64 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	// An exact-limit input reaches document validation; this oversized zero tail
	// is unsupported, but it must not be misclassified as an input-budget error.
	if status := cli.Run([]string{"export", input, "--format", "text", "--output", output}, &out, &diagnostic); status != 1 || out.Len() != 0 || strings.Contains(diagnostic.String(), "resource_limit at byte 0 (input)") || diagnostic.Len() == 0 {
		t.Fatalf("inclusive input budget: %d %q %q", status, out.String(), diagnostic.String())
	}
	if err = os.Truncate(input, 64*1024*1024+1); err != nil {
		t.Fatal(err)
	}
	diagnostic.Reset()
	if status := cli.Run([]string{"export", input, "--format", "text", "--output", output}, &out, &diagnostic); status != 1 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "resource_limit at byte 0 (input)") {
		t.Fatalf("oversize: %d %q %q", status, out.String(), diagnostic.String())
	}
	if _, err = os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("oversize publication: %v", err)
	}
}
