package apply

import (
	"cloudsave/pkg/data"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	ListCmd struct {
		Service *data.Service
	}
)

func (*ListCmd) Name() string     { return "apply" }
func (*ListCmd) Synopsis() string { return "apply a backup" }
func (*ListCmd) Usage() string {
	return `Usage: cloudsave apply <GAME_ID> [BACKUP_ID]

Apply a backup
`
}

func (p *ListCmd) SetFlags(f *flag.FlagSet) {
}

func (p *ListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: missing game ID and/or backup uuid")
		return subcommands.ExitUsageError
	}

	gameID := f.Arg(0)
	uuid := f.Arg(1)

	if len(uuid) == 0 {
		if err := p.Service.ApplyCurrent(gameID); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to apply: %s", err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	if err := p.Service.ApplyBackup(gameID, uuid); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to apply: %s", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
