package remove

import (
	"cloudsave/pkg/data"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	RemoveCmd struct {
		Service *data.Service
	}
)

func (*RemoveCmd) Name() string     { return "remove" }
func (*RemoveCmd) Synopsis() string { return "unregister a game" }
func (*RemoveCmd) Usage() string {
	return `Usage: cloudsave remove <GAME_ID>

Unregister a game
Caution: all the backup are deleted
`
}

func (p *RemoveCmd) SetFlags(f *flag.FlagSet) {
}

func (p *RemoveCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: the command is expecting for 1 argument")
		return subcommands.ExitUsageError
	}

	err := p.Service.RemoveGame(f.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to unregister the game:", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
