// Package main is the entry point for the upp CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/JhnFrankz/upp/internal/cli"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	root, gf := cli.BuildRoot()
	root.Version = version
	cli.AddCommands(root, gf)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
