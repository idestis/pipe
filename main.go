package main

import "github.com/getpipe-dev/pipe/internal/cli"

// version is set by goreleaser via ldflags
var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
