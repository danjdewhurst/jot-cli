package main

import (
	"os"

	"github.com/danjdewhurst/jot-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
