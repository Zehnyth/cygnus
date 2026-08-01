// Cygnus is a Linux host memory-triage CLI. This file is the binary's
// entry point: it hands off to the cli package and does nothing else.
package main

import (
	"fmt"
	"os"

	"github.com/zehnyth/cygnus/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "cygnus:", err)
		os.Exit(1)
	}
}
