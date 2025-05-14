package main

import (
	"cloudsave/pkg/tools/windows"
	"os"
)

const defaultDocumentRoot string = "C:/ProgramData/CloudSave"

func main() {
	run()
}

func fatal(message string, exitCode int) {
	windows.MessageBox(windows.NULL, message, "CloudSave", windows.MB_OK)
	os.Exit(exitCode)
}
