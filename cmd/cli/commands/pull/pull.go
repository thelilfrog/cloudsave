package pull

import (
	"cloudsave/cmd/cli/tools/prompt/credentials"
	"cloudsave/pkg/remote/client"
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
	PullCmd struct {
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
		fmt.Fprintf(os.Stderr, "failed to read std output: %s", err)
		return subcommands.ExitFailure
	}

	cli := client.New(url, username, password)

	if err := cli.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to the remote: %s", err)
		return subcommands.ExitFailure
	}

	archivePath := filepath.Join(repository.DatastorePath(), gameID, "data.tar.gz")

	m, err := cli.Metadata(gameID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get metadata: %s", err)
		return subcommands.ExitFailure
	}

	err = repository.Register(m, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to register local metadata: %s", err)
		return subcommands.ExitFailure
	}

	if err := cli.Pull(gameID, archivePath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to pull from the remote: %s", err)
		return subcommands.ExitFailure
	}

	fi, err := os.OpenFile(archivePath, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open archive: %s", err)
		return subcommands.ExitFailure
	}

	if err := archive.Untar(fi, path); err != nil {
		fmt.Fprintf(os.Stderr, "failed to unarchive file: %s", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
