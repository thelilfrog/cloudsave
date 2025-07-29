package version

import (
	"cloudsave/pkg/constants"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/tools/prompt/credentials"
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/google/subcommands"
)

type (
	VersionCmd struct {
		remote bool
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
	f.BoolVar(&p.remote, "a", false, "get a remote version information")
}

func (p *VersionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.remote {
		if f.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "error: missing remote url")
			return subcommands.ExitUsageError
		}

		username, password, err := credentials.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read std output: %s", err)
			return subcommands.ExitFailure
		}

		if err := remote(f.Arg(0), username, password); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}
	local()
	return subcommands.ExitSuccess
}

func local() {
	fmt.Println("Client: CloudSave cli")
	fmt.Println(" Version:       " + constants.Version)
	fmt.Println(" API version:   " + strconv.Itoa(constants.ApiVersion))
	fmt.Println(" Go version:    " + runtime.Version())
	fmt.Println(" OS/Arch:       " + runtime.GOOS + "/" + runtime.GOARCH)
}

func remote(url, username, password string) error {
	cli := client.New(url, username, password)

	if err := cli.Ping(); err != nil {
		return fmt.Errorf("failed to connect to the remote: %w", err)
	}

	info, err := cli.Version()
	if err != nil {
		return fmt.Errorf("failed to load games from remote: %w", err)
	}

	fmt.Println()
	fmt.Println("Remote:", url)
	fmt.Println("---")
	fmt.Println("Server:")
	fmt.Println(" Version:       " + info.Version)
	fmt.Println(" API version:   " + strconv.Itoa(info.APIVersion))
	fmt.Println(" Go version:    " + info.GoVersion)
	fmt.Println(" OS/Arch:       " + info.OSName + "/" + info.OSArchitecture)

	return nil
}
