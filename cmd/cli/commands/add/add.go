package add

import (
	"cloudsave/pkg/game"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
)

type (
	AddCmd struct {
		name string
	}
)

func (AddCmd) Name() string     { return "add" }
func (AddCmd) Synopsis() string { return "Add a folder to the sync list" }
func (AddCmd) Usage() string {
	return `add:
  Add a folder to the sync list
`
}

func (p AddCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.name, "name", "", "Override the name of the game")
}

func (p AddCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
		p.name = filepath.Base(filepath.Dir(path))
	}

	m, err := game.Add(p.name, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to add game reference:", err)
		return subcommands.ExitFailure
	}

	fmt.Println(m.ID)

	return subcommands.ExitSuccess
}
