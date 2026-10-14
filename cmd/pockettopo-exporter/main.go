package main

import (
	"os"

	"pockettopo-exporter/internal/cli"
)

func main() {
	if status, handled := runDesktop(os.Args[1:], os.Stdout, os.Stderr); handled {
		os.Exit(status)
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
