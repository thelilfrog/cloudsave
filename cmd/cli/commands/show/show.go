package show

import (
	"cloudsave/pkg/data"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	ShowCmd struct {
		Service *data.Service
	}
)

func (*ShowCmd) Name() string     { return "show" }
func (*ShowCmd) Synopsis() string { return "show metadata about game" }
func (*ShowCmd) Usage() string {
	return `Usage: cloudsave show <GAME_ID>

Show metdata about a game
`
}

func (p *ShowCmd) SetFlags(f *flag.FlagSet) {
}

func (p *ShowCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: missing game ID")
		return subcommands.ExitUsageError
	}

	gameID := f.Arg(0)
	g, err := p.Service.One(gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to apply: %s", err)
		return subcommands.ExitFailure
	}

	fmt.Println(g.Name)
	fmt.Println("------")
	fmt.Println("Version: ", g.Version)
	fmt.Println("Path: ", g.Path)
	fmt.Println("MD5: ", g.MD5)

	return subcommands.ExitSuccess
}
