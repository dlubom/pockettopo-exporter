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

func TestGeometryCommand(t *testing.T) {
	path := "../processing/testdata/geometry/chain.top"
	for _, close := range []bool{false, true} {
		args := []string{"geometry", path, "--json"}
		if close {
			args = append(args, "--close-loops")
		}
		var out, err bytes.Buffer
		if code := cli.Run(args, &out, &err); code != 0 || err.Len() != 0 {
			t.Fatalf("%d %s", code, err.String())
		}
		original, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		hash := sha256.Sum256(original)
		if !strings.Contains(out.String(), hex.EncodeToString(hash[:])) {
			t.Fatal("source hash lost")
		}
		var v struct {
			Schema  int    `json:"schema_version"`
			Profile string `json:"profile"`
			Closed  bool   `json:"closed"`
			Nodes   []struct {
				Key, Status string
				Position    struct {
					X int32 `json:"x_mm"`
				}
			} `json:"nodes"`
		}
		if e := json.Unmarshal(out.Bytes(), &v); e != nil {
			t.Fatal(e)
		}
		if v.Schema != 1 || v.Profile != "isolated_native_geometry" || v.Closed != close || len(v.Nodes) != 7 || v.Nodes[1].Key != "group:0" || v.Nodes[1].Position.X != 354 {
			t.Fatalf("%+v", v)
		}
		var again bytes.Buffer
		if cli.Run(args, &again, &err) != 0 || !bytes.Equal(out.Bytes(), again.Bytes()) {
			t.Fatal("nondeterministic geometry")
		}
	}
}
func TestGeometryCommandFailures(t *testing.T) {
	for _, args := range [][]string{{"geometry"}, {"geometry", "", "--json"}, {"geometry", "--json", "--json"}, {"geometry", "file", "x"}, {"geometry", "file", "--json", "bad"}, {"geometry", "file", "--json", "--close-loops", "bad"}, {"geometry", "missing", "--json"}, {"geometry", ".", "--json"}, {"geometry", "../processing/testdata/geometry/failure.top", "--json"}} {
		var out, err bytes.Buffer
		if code := cli.Run(args, &out, &err); code != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatalf("%v %d %s %s", args, code, &out, &err)
		}
	}
	for _, b := range [][]byte{[]byte("bad"), nil} {
		p := filepath.Join(t.TempDir(), "bad.top")
		if e := os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
		var out, err bytes.Buffer
		if cli.Run([]string{"geometry", p, "--json"}, &out, &err) != 1 || out.Len() != 0 || !strings.Contains(err.String(), "Cannot process") {
			t.Fatal("bad source", err.String())
		}
	}
	var err bytes.Buffer
	if cli.Run([]string{"geometry", "../processing/testdata/geometry/chain.top", "--json"}, failingWriter{}, &err) != 1 || err.Len() == 0 {
		t.Fatal("output error")
	}
}
