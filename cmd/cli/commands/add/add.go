package add

import (
	"cloudsave/pkg/data"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
)

type (
	AddCmd struct {
		Service *data.Service
		name    string
		remote  string
	}
)

func (*AddCmd) Name() string     { return "add" }
func (*AddCmd) Synopsis() string { return "add a folder to the sync list" }
func (*AddCmd) Usage() string {
	return `Usage: cloudsave add [-name] [-remote] <PATH>

Add a folder to the track list
	
Options:
`
}

func (p *AddCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.name, "name", "", "Override the name of the game")
	f.StringVar(&p.remote, "remote", "", "Defines a remote server to sync with")
}

func (p *AddCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: the command is expecting for 1 argument")
		return subcommands.ExitUsageError
	}
	path, err := filepath.Abs(f.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot get the absolute path for this entry:", err)
		return subcommands.ExitFailure
	}

	if p.name == "" {
		p.name = filepath.Base(path)
	}

	gameID, err := p.Service.Add(p.name, path, p.remote)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to add this gamesave to the datastore:", err)
		return subcommands.ExitFailure
	}

	if err := p.Service.Scan(gameID); err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to scan:", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
