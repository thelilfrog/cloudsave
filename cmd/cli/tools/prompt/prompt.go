package prompt

import (
	"fmt"
	"strings"
)

type (
	ConflictResponse int
)

const (
	My ConflictResponse = iota
	Their
	Abort
)

func ScanBool(msg string, defaultValue bool) bool {
	fmt.Printf("%s: ", msg)

	var r string
	if _, err := fmt.Scanln(&r); err != nil {
		panic(err)
	}

	return strings.ToLower(r) == "y"
}

func Conflict() ConflictResponse {
	fmt.Print("[M: My, T: Their, A: Abort]: ")

	var r string
	if _, err := fmt.Scanln(&r); err != nil {
		panic(err)
	}

	switch strings.ToLower(r) {
	case "m":
		return My
	case "t":
		return Their
	default:
		return Abort
	}
}
