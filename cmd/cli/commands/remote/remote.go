package remote

import (
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
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

func (p *RemoteCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	switch {
	case p.list:
		{
			if err := list(); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return subcommands.ExitFailure
			}
		}
	case p.set:
		{
			if f.NArg() != 2 {
				subcommands.HelpCommand().Execute(ctx, f, nil)
				return subcommands.ExitUsageError
			}
			if err := set(f.Arg(0), f.Arg(1)); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return subcommands.ExitFailure
			}
		}
	default:
		{
			subcommands.HelpCommand().Execute(ctx, f, nil)
			return subcommands.ExitUsageError
		}
	}
	return subcommands.ExitSuccess
}

func list() error {
	games, err := game.All()
	if err != nil {
		return fmt.Errorf("failed to load datastore: %w", err)
	}

	for _, g := range games {
		r, err := remote.One(g.ID)
		if err != nil {
			return fmt.Errorf("failed to load datastore: %w", err)
		}

		cli := client.New(r.URL, "", "")

		status := "OK"
		if err := cli.Ping(); err != nil {
			status = "ERROR: " + err.Error()
		}

		fmt.Printf("'%s' -> %s (%s)\n", g.Name, r.URL, status)
	}

	return nil
}

func set(gameID, url string) error {
	return remote.Set(gameID, url)
}
