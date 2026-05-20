package main

import (
	"fmt"
	"os"

	"github.com/ntts9990/ntts-flightlog/internal/cli"
)

// version is injected at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
