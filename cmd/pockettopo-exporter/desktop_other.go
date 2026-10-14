//go:build !windows

package main

import (
	"fmt"
	"io"
)

func runDesktop(args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) > 0 && (args[0] == "windows-menu" || args[0] == "shell-export") {
		fmt.Fprintln(stderr, "Windows Explorer integration is supported only on Windows. Use export for portable noninteractive exports.")
		return 1, true
	}
	return 0, false
}
