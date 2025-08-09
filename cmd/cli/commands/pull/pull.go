package pull

import (
	"cloudsave/cmd/cli/tools/prompt/credentials"
	"cloudsave/pkg/data"
	"cloudsave/pkg/remote/client"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	PullCmd struct {
		Service *data.Service
	}
)

func (*PullCmd) Name() string     { return "pull" }
func (*PullCmd) Synopsis() string { return "pull a game save from the remote" }
func (*PullCmd) Usage() string {
	return `Usage: cloudsave pull <GAME_ID>

Pull a game save from the remote
`
}

func (p *PullCmd) SetFlags(f *flag.FlagSet) {

}

func (p *PullCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "error: missing arguments")
		return subcommands.ExitUsageError
	}

	url := f.Arg(0)
	gameID := f.Arg(1)
	path := f.Arg(2)

	username, password, err := credentials.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read std output: %s", err)
		return subcommands.ExitFailure
	}

	cli := client.New(url, username, password)

	if err := cli.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to the remote: %s", err)
		return subcommands.ExitFailure
	}

	if err := p.Service.PullCurrent(gameID, path, cli); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to pull current archive: %s", err)
		return subcommands.ExitFailure
	}

	ids, err := cli.ListArchives(gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to list backup archive: %s", err)
		return subcommands.ExitFailure
	}

	for _, id := range ids {
		if err := p.Service.PullBackup(gameID, id, cli); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to pull backup archive %s: %s", id, err)
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}
