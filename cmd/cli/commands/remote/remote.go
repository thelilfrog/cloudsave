package remote

import (
	"cloudsave/pkg/data"
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
		Service *data.Service
		set     bool
		list    bool
	}
)

func (*RemoteCmd) Name() string     { return "remote" }
func (*RemoteCmd) Synopsis() string { return "add or update the remote url" }
func (*RemoteCmd) Usage() string {
	return `Usage: cloudsave remote <-set|-list>

The -list argument lists all remotes for each registered game.
This command performs a connection test.

The -set argument allow you to set (create or update) 
the URL to the remote for a game

Options
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
			if err := p.print(); err != nil {
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

func (p *RemoteCmd) print() error {
	games, err := p.Service.AllGames()
	if err != nil {
		return fmt.Errorf("failed to load datastore: %w", err)
	}

	for _, g := range games {
		r, err := remote.One(g.ID)
		if err != nil {
			continue
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
