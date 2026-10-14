package exportsvx

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Omitting either lexical reset, resetting it after the first comment, or
// leaking it outside the anonymous scope must fail with the real consumer.
func TestSVXInheritedLexicalSettings(t *testing.T) {
	cavern, e := exec.LookPath("cavern")
	if e != nil {
		t.Skip("optional Survex consumer; the required CI consumer gate covers this case")
	}
	banner, e := exec.Command(cavern, "--version").CombinedOutput()
	if e != nil {
		t.Fatalf("consumer version: %v\n%s", e, banner)
	}
	if !strings.Contains(string(banner), "Survex 1.4.22") && !strings.Contains(string(banner), "Survex 1.4.23") {
		t.Skipf("consumer regression requires Survex 1.4.22 or 1.4.23: %s", banner)
	}
	r, e := Build(document(t, "features"))
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, prefix, suffix string
	}{
		{"decimal-comma", "*set blank x09x20\n*set decimal ,\n", "1.4 after1 1,25 90,0 0,0\n"},
		{"hash-comments", "*set comment #\n", "1.4 after1 1.25 90 0 # restored comment\n"},
		{"combined", "*set blank x09x20\n*set decimal ,\n*set comment #\n", "1.4 after1 1,25 90,0 0,0 # restored comment\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if e := os.WriteFile(filepath.Join(dir, "survey.svx"), r.Text, 0600); e != nil {
				t.Fatal(e)
			}
			host := "*set separator :\n*set names ._-\n" + tc.prefix + "*include survey.svx\n*data normal from to tape compass clino\n" + tc.suffix
			if e := os.WriteFile(filepath.Join(dir, "host.svx"), []byte(host), 0600); e != nil {
				t.Fatal(e)
			}
			cmd := exec.Command(cavern, "-o", "host.3d", "host.svx")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "SURVEXLANG=en")
			log, e := cmd.CombinedOutput()
			if e != nil {
				t.Fatalf("SVX include or caller settings restoration failed: %v\n%s", e, log)
			}
			// Seven feature legs (including one splay), plus the caller's 1.25 m
			// leg. These literal statistics also reject silently ignored data.
			for _, pattern := range []string{`joined by 8 legs`, `There are 0 loops`, `Total length of survey legs =\s+22\.25m`} {
				if !regexp.MustCompile(pattern).Match(log) {
					t.Fatalf("missing consumer statistic %s\n%s", pattern, log)
				}
			}
		})
	}
}
