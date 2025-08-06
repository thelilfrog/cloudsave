package apply

import (
	"cloudsave/pkg/repository"
	"cloudsave/pkg/tools/archive"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

	gameID := f.Arg(0)
	uuid := f.Arg(1)

	g, err := repository.One(gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open game metadata: %s\n", err)
		return subcommands.ExitFailure
	}

	if err := repository.RestoreArchive(gameID, uuid); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to restore backup: %s\n", err)
		return subcommands.ExitFailure
	}

	if err := os.RemoveAll(g.Path); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to remove old data: %s\n", err)
		return subcommands.ExitFailure
	}

	file, err := os.OpenFile(filepath.Join(repository.DatastorePath(), gameID, "data.tar.gz"), os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open archive: %s\n", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	if err := archive.Untar(file, g.Path); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to extract archive: %s\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
