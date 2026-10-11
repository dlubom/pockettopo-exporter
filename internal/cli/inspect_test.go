package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pockettopo-exporter/internal/cli"
)

func inspectFile(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "input.top")
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func inspectBytes() []byte {
	b := append([]byte{'T', 'o', 'p', 3}, make([]byte, 36)...)
	b = append(b, 0)
	b = append(b, make([]byte, 12)...)
	return append(b, 0, 0, 0, 0, 0)
}
func TestInspectJSON(t *testing.T) {
	b := inspectBytes()
	path := inspectFile(t, b)
	hash := sha256.Sum256(b)
	expected := `{"schema_version":1,"tool_version":"dev","source":{"sha256":"` + hex.EncodeToString(hash[:]) + `","size_bytes":58,"top_version":3},"status":"supported","complete":true,"supported":true,"counts":{"trips":0,"measurements":0,"references":0},"plan":{"elements":0,"polygons":0,"xsections":0,"points":0,"stop_reason":"terminated","stop_marker":0,"stop_offset":40},"side":{"elements":0,"polygons":0,"xsections":0,"points":0,"stop_reason":"terminated","stop_marker":0,"stop_offset":53},"accounting":{"body_end":54,"consumed_bytes":58,"unparsed_bytes":0,"tail_kind":"zero_trailer"},"diagnostics":[]}` + "\n"
	for i := 0; i < 2; i++ {
		var out, err bytes.Buffer
		if status := cli.Run([]string{"inspect", path, "--json"}, &out, &err); status != 0 || err.Len() != 0 || out.String() != expected {
			t.Fatalf("got %d %s %s want %s", status, &out, &err, expected)
		}
	}
	after, e := os.ReadFile(path)
	if e != nil || !bytes.Equal(after, b) {
		t.Fatal("input changed")
	}
}
func TestInspectAnalysisFailures(t *testing.T) {
	for _, tc := range []struct {
		name             string
		b                []byte
		status, code     string
		offset, consumed int
		complete         bool
	}{
		{"partial", inspectBytes()[:55], "unsupported", "unsupported_tail", 54, 54, true},
		{"nonzero", append(inspectBytes()[:54], 1, 0, 0, 0), "unsupported", "unsupported_tail", 54, 54, true},
		{"extra", append(inspectBytes(), 255), "unsupported", "unsupported_tail", 58, 58, true},
		{"missing side stop", inspectBytes()[:53], "malformed", "truncated", 53, 0, false},
		{"bad magic", []byte("BAD\x03"), "malformed", "bad_magic", 0, 0, false},
		{"version", []byte("Top\x02"), "unsupported", "unsupported_version", 3, 0, false},
		{"empty", nil, "malformed", "truncated", 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			if cli.Run([]string{"inspect", inspectFile(t, tc.b), "--json"}, &out, &err) != 1 || err.Len() == 0 {
				t.Fatal("exit")
			}
			var s struct {
				Status     string
				Complete   bool
				Supported  bool
				Accounting struct {
					Consumed int `json:"consumed_bytes"`
					Unparsed int `json:"unparsed_bytes"`
				}
				Diagnostics []struct {
					Code   string
					Offset int
				}
			}
			if e := json.Unmarshal(out.Bytes(), &s); e != nil {
				t.Fatal(e)
			}
			if s.Status != tc.status || s.Complete != tc.complete || s.Supported || s.Accounting.Consumed != tc.consumed || s.Accounting.Unparsed != len(tc.b)-tc.consumed || len(s.Diagnostics) != 1 || s.Diagnostics[0].Code != tc.code || s.Diagnostics[0].Offset != tc.offset {
				t.Fatalf("%s", &out)
			}
		})
	}
	for _, offset := range []int{40, 53} {
		b := inspectBytes()
		b[offset] = 255
		var out, err bytes.Buffer
		if cli.Run([]string{"inspect", inspectFile(t, b), "--json"}, &out, &err) != 1 || !strings.Contains(out.String(), `"code":"unsupported_element"`) || !strings.Contains(out.String(), `"complete":false`) {
			t.Fatal(&out, &err)
		}
	}
	b := inspectBytes()[:54]
	var out, err bytes.Buffer
	if cli.Run([]string{"inspect", inspectFile(t, b), "--json"}, &out, &err) != 0 || !strings.Contains(out.String(), `"tail_kind":"absent"`) {
		t.Fatal(&out, &err)
	}
}
func TestInspectRecords(t *testing.T) {
	var out, err bytes.Buffer
	if cli.Run([]string{"inspect", "../top/testdata/api-drawings.top", "--json"}, &out, &err) != 0 {
		t.Fatal(&err)
	}
	if !strings.Contains(out.String(), `"counts":{"trips":1,"measurements":2,"references":0}`) || !strings.Contains(out.String(), `"side":{"elements":10,"polygons":8,"xsections":2,"points":22`) {
		t.Fatal(&out)
	}
}
func TestInspectIOAndArguments(t *testing.T) {
	path := inspectFile(t, inspectBytes())
	for _, args := range [][]string{{"inspect"}, {"inspect", path}, {"inspect", path, "--xml"}, {"inspect", path, "--json", "extra"}, {"inspect", "", "--json"}, {"inspect", "--json", path}, {"inspect", filepath.Join(t.TempDir(), "missing"), "--json"}, {"inspect", t.TempDir(), "--json"}} {
		var out, err bytes.Buffer
		if cli.Run(args, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal(args, &out, &err)
		}
	}
	var err bytes.Buffer
	if cli.Run([]string{"inspect", path, "--json"}, failingWriter{}, &err) != 1 || !strings.Contains(err.String(), "Cannot write output") {
		t.Fatal(&err)
	}
	large := filepath.Join(t.TempDir(), "large.top")
	f, e := os.Create(large)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(64*1024*1024 + 1); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	err.Reset()
	if cli.Run([]string{"inspect", large, "--json"}, &out, &err) != 1 || out.Len() != 0 || !strings.Contains(err.String(), "resource_limit") {
		t.Fatal(&out, &err)
	}
}

func TestInspectHeaderBoundary(t *testing.T) {
	var out, err bytes.Buffer
	cli.Run([]string{"inspect", inspectFile(t, []byte("Top\x03")), "--json"}, &out, &err)
	if !strings.Contains(out.String(), `"top_version":3`) {
		t.Fatal(&out)
	}
	out.Reset()
	err.Reset()
	cli.Run([]string{"inspect", inspectFile(t, []byte("Top")), "--json"}, &out, &err)
	if !strings.Contains(out.String(), `"top_version":null`) {
		t.Fatal(&out)
	}
}

func TestInspectExactInputLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exact-limit.top")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(64 * 1024 * 1024); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	var out, err bytes.Buffer
	if cli.Run([]string{"inspect", path, "--json"}, &out, &err) != 1 || !strings.Contains(out.String(), `"size_bytes":67108864`) || !strings.Contains(out.String(), `"code":"bad_magic"`) {
		t.Fatal("inclusive input limit", &out, &err)
	}
}
