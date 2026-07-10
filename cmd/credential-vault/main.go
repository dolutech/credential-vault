// Package main is the entry point for the credential-vault CLI.
package main

import (
	"fmt"
	"os"

	"credential-vault/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}