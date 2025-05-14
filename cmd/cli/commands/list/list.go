package list

import (
	"cloudsave/pkg/game"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	ListCmd struct {
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
}

func (p *ListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	datastore, err := game.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	fmt.Println("ID | NAME | PATH")
	fmt.Println("-- | ---- | ----")
	for _, metadata := range datastore {
		fmt.Println(metadata.ID, "|", metadata.Name, "|", metadata.Path)
	}

	return subcommands.ExitSuccess
}
