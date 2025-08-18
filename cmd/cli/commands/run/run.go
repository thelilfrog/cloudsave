package run

import (
	"cloudsave/pkg/data"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	RunCmd struct {
		Service *data.Service
	}
)

func (*RunCmd) Name() string     { return "scan" }
func (*RunCmd) Synopsis() string { return "check and process all the folder" }
func (*RunCmd) Usage() string {
	return `Usage: cloudsave scan

Check if the files have been modified. If so,
the current archive is moved to the backup list
and a new archive is created with a new version number. 
`
}

func (p *RunCmd) SetFlags(f *flag.FlagSet) {}

func (p *RunCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	datastore, err := p.Service.AllGames()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	for _, metadata := range datastore {
		changed, err := p.Service.Scan(metadata.ID)
		if err != nil {
			fmt.Println("❌", metadata.Name, ":", err.Error())
			continue
		}
		if changed {
			fmt.Println("✅", metadata.Name, ": backed up")
		} else {
			fmt.Println("🆗", metadata.Name, ": up to date")
		}
	}

	fmt.Println("done.")
	return subcommands.ExitSuccess
}
