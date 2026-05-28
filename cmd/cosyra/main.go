package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "on", "off", "status", "hook":
		fmt.Fprintf(os.Stderr, "cosyra %s is not implemented yet\n", os.Args[1])
		os.Exit(1)
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: cosyra <on|off|status|hook|version>\n")
}
