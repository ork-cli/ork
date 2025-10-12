package main

import (
	"github.com/ork-cli/ork/internal/cli"
)

// Build information. Populated at build-time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	cli.Execute()
}
