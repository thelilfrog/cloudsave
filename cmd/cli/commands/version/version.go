package version

import (
	"cloudsave/pkg/constants"
	"context"
	"flag"
	"fmt"
	"runtime"
	"strconv"

	"github.com/google/subcommands"
)

type (
	VersionCmd struct {
	}
)

func (*VersionCmd) Name() string     { return "version" }
func (*VersionCmd) Synopsis() string { return "show version and system information" }
func (*VersionCmd) Usage() string {
	return `add:
  Show version and system information
`
}

func (p *VersionCmd) SetFlags(f *flag.FlagSet) {
}

func (p *VersionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fmt.Println("Client: CloudSave cli")
	fmt.Println(" Version:       " + constants.Version)
	fmt.Println(" API version:   " + strconv.Itoa(constants.ApiVersion))
	fmt.Println(" Go version:    " + runtime.Version())
	fmt.Println(" OS/Arch:       " + runtime.GOOS + "/" + runtime.GOARCH)

	return subcommands.ExitSuccess
}
