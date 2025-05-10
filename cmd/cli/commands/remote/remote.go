package remote

import (
	"cloudsave/pkg/remote"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	RemoteCmd struct {
		set  bool
		list bool
	}
)

func (*RemoteCmd) Name() string     { return "remote" }
func (*RemoteCmd) Synopsis() string { return "manage remote" }
func (*RemoteCmd) Usage() string {
	return `remote:
  manage remove
`
}

func (p *RemoteCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.list, "list", false, "list remotes")
	f.BoolVar(&p.set, "set", false, "set remote for a game")
}

func (p *RemoteCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.list {
		remotes, err := remote.All()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
			return subcommands.ExitFailure
		}

		fmt.Println("ID | REMOTE URL")
		fmt.Println("-- | ----------")
		for _, remote := range remotes {
			fmt.Println(remote.GameID, "|", remote.URL)
		}
		return subcommands.ExitSuccess
	}

	if p.set {
		if f.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "error: the command is expecting for 2 arguments")
			f.Usage()
			return subcommands.ExitUsageError
		}

		err := remote.Set(f.Arg(0), f.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to set remote:", err)
			return subcommands.ExitFailure
		}
		fmt.Println(f.Arg(0))
		return subcommands.ExitSuccess
	}

	f.Usage()
	return subcommands.ExitUsageError
}
