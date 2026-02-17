package main

import "github.com/idestis/pipe/internal/cli"

// version is set by goreleaser via ldflags
var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
