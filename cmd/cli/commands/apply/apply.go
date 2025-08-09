package apply

import (
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

func (*ListCmd) Name() string     { return "apply" }
func (*ListCmd) Synopsis() string { return "apply a backup" }
func (*ListCmd) Usage() string {
	return `Usage: cloudsave apply <GAME_ID> <BACKUP_ID>

Apply a backup
`
}

func (p *ListCmd) SetFlags(f *flag.FlagSet) {
}

func (p *ListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "error: missing game ID and/or backup uuid")
		return subcommands.ExitUsageError
	}

	//gameID := f.Arg(0)
	//uuid := f.Arg(1)

	panic("not implemented")

	return subcommands.ExitSuccess
}
