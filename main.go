package main

import "github.com/dominionthedev/logfmt/cmd"

// version is set at build time via -ldflags "-X main.version=v1.0.0"
var version = "dev"

func main() {
	cmd.Execute(version)
}