package cli_test

import (
	"bytes"
	"encoding/json"
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
