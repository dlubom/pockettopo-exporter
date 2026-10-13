package cli

import (
	"fmt"
	"io"
	"pockettopo-exporter/internal/exportwalls"
	"pockettopo-exporter/internal/top"
	"strings"
)

func runWalls(args []string, stdout, stderr io.Writer) int {
	if len(args) != 6 || args[1] == "" || strings.HasPrefix(args[1], "--") || args[4] != "--output" || args[5] == "" {
		fmt.Fprint(stderr, help)
		return 1
	}
	b, e := readContextFile(args[1], top.DefaultDocumentLimits().MaxInputBytes)
	if e != nil {
		return exportError(stderr, e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		return exportError(stderr, e)
	}
	r, e := exportwalls.Build(d)
	if e != nil {
		return exportError(stderr, e)
	}
	if e = publishNamedResult(args[5], publicationResult{r.Text, r.Report}, "survey.srv"); e != nil {
		return exportError(stderr, e)
	}
	if _, e = io.WriteString(stdout, "Exported survey.srv and report.json\n"); e != nil {
		return exportError(stderr, fmt.Errorf("export published; cannot write notification: %w", e))
	}
	return 0
}
