package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/cli"
	"strings"
	"testing"
)

func TestProcessNativeGrouping(t *testing.T) {
	path := "../processing/testdata/repeats.top"
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(b)
	var out, err bytes.Buffer
	if code := cli.Run([]string{"process", path, "--json"}, &out, &err); code != 0 || err.Len() != 0 {
		t.Fatalf("code=%d %s", code, &err)
	}
	var v struct {
		Schema int `json:"schema_version"`
		Source struct {
			SHA  string `json:"sha256"`
			Size int    `json:"size_bytes"`
		}
		Rows   []json.RawMessage
		Groups []struct {
			Head    int `json:"head_index"`
			Members []struct {
				Row     int  `json:"row_index"`
				Reverse bool `json:"reversed"`
			}
			Vector struct {
				DX int32 `json:"dx_mm"`
			}
			Forward struct {
				Bits [3]uint32 `json:"float32_bits"`
			}
		}
	}
	if e := json.Unmarshal(out.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if v.Schema != 1 || v.Source.SHA != hex.EncodeToString(hash[:]) || v.Source.Size != len(b) || len(v.Rows) != 12 || len(v.Groups) != 5 {
		t.Fatalf("%+v", v)
	}
	g := v.Groups[0]
	if g.Head != 0 || len(g.Members) != 3 || g.Members[2].Row != 2 || !g.Members[2].Reverse || g.Vector.DX != -290 || g.Forward.Bits != [3]uint32{0x43ddfa3c, 0x40ae347a, 0x3e14227d} {
		t.Fatal(g)
	}
	golden, e := os.ReadFile("testdata/process-repeats.json")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(out.Bytes(), golden) {
		t.Fatal("process JSON changed")
	}
	var second bytes.Buffer
	if code := cli.Run([]string{"process", path, "--json"}, &second, &err); code != 0 || !bytes.Equal(out.Bytes(), second.Bytes()) {
		t.Fatal("nondeterministic", code)
	}
	if after, e := os.ReadFile(path); e != nil || !bytes.Equal(after, b) {
		t.Fatal("source changed", e)
	}
}
func TestProcessFailuresAndEmpty(t *testing.T) {
	for _, args := range [][]string{{"process"}, {"process", "x", "--json", "extra"}, {"process", "", "--json"}, {"process", "--json", "--json"}, {"process", "x", "--other"}, {"process", "missing.top", "--json"}} {
		var out, err bytes.Buffer
		if code := cli.Run(args, &out, &err); code != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatalf("%q code=%d out=%s err=%s", args, code, &out, &err)
		}
	}
	dir := t.TempDir()
	for _, data := range [][]byte{nil, []byte("Top\x04"), append(mustRead(t, "../processing/testdata/empty.top"), 1)} {
		path := filepath.Join(dir, "bad.top")
		if e := os.WriteFile(path, data, 0600); e != nil {
			t.Fatal(e)
		}
		var out, err bytes.Buffer
		if code := cli.Run([]string{"process", path, "--json"}, &out, &err); code != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatalf("code=%d out=%s err=%s", code, &out, &err)
		}
	}
	var out, err bytes.Buffer
	if code := cli.Run([]string{"process", "../processing/testdata/empty.top", "--json"}, &out, &err); code != 0 || err.Len() != 0 || !strings.Contains(out.String(), `"rows":[],"groups":[]`) {
		t.Fatalf("code=%d %s %s", code, &out, &err)
	}
	err.Reset()
	if code := cli.Run([]string{"process", "../processing/testdata/repeats.top", "--json"}, failingWriter{}, &err); code != 1 || !strings.Contains(err.String(), "Cannot process:") {
		t.Fatal(code, &err)
	}
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestProcessInputLimitsAndReadFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.top")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(64*1024*1024 + 1); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	var out, err bytes.Buffer
	if code := cli.Run([]string{"process", path, "--json"}, &out, &err); code != 1 || out.Len() != 0 || !strings.Contains(err.String(), "resource_limit at byte 0") {
		t.Fatal(code, &out, &err)
	}
	err.Reset()
	if code := cli.Run([]string{"process", dir, "--json"}, &out, &err); code != 1 || out.Len() != 0 || err.Len() == 0 {
		t.Fatal(code, &out, &err)
	}
}
