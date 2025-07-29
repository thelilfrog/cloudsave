package list

import (
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/tools/prompt/credentials"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	ListCmd struct {
		remote bool
	}
)

func (*ListCmd) Name() string     { return "list" }
func (*ListCmd) Synopsis() string { return "list all game registered" }
func (*ListCmd) Usage() string {
	return `list:
  List all game registered
`
}

func (p *ListCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.remote, "a", false, "list all including remote data")
}

func (p *ListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
	if err := local(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func local() error {
	games, err := game.All()
	if err != nil {
		return fmt.Errorf("failed to load datastore: %w", err)
	}

	for _, g := range games {
		fmt.Println("ID:", g.ID)
		fmt.Println("Name:", g.Name)
		fmt.Println("Last Version:", g.Date, "( Version Number", g.Version, ")")
		fmt.Println("---")
	}

	return nil
}

func remote(url, username, password string) error {
	cli := client.New(url, username, password)

	if err := cli.Ping(); err != nil {
		return fmt.Errorf("failed to connect to the remote: %w", err)
	}

	games, err := cli.All()
	if err != nil {
		return fmt.Errorf("failed to load games from remote: %w", err)
	}

	fmt.Println()
	fmt.Println("Remote:", url)
	fmt.Println("---")
	for _, g := range games {
		fmt.Println("ID:", g.ID)
		fmt.Println("Name:", g.Name)
		fmt.Println("Last Version:", g.Date, "( Version Number", g.Version, ")")
		fmt.Println("---")
	}

	return nil
}
