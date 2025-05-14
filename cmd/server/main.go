//go:build !windows

package main

import (
	"fmt"
	"os"
)

const defaultDocumentRoot string = "/var/lib/cloudsave"

func main() {
	run()
}

func fatal(message string, exitCode int) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(exitCode)
}
