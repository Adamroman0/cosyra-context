package main

import (
	"fmt"
	"os"

	"github.com/adamroman0/cosyra-context/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Run(os.Args[1:], version, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
