package sync

import (
	"cloudsave/pkg/remote"
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/google/subcommands"
)

type (
	SyncCmd struct {
	}
)

func (*SyncCmd) Name() string     { return "sync" }
func (*SyncCmd) Synopsis() string { return "list all game registered" }
func (*SyncCmd) Usage() string {
	return `add:
  List all game registered
`
}

func (p *SyncCmd) SetFlags(f *flag.FlagSet) {
}

func (p *SyncCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	_, err := remote.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	for _, remote := range remotes {
		
	}

	return subcommands.ExitSuccess
}

func hash(path string) string {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		notFound("id not found", w, r)
		return
	}
	defer f.Close()

	// Create MD5 hasher
	hasher := md5.New()

	// Copy file content into hasher
	if _, err := io.Copy(hasher, f); err != nil {
		fmt.Fprintln(os.Stderr, "error: an error occured while reading data:", err)
		internalServerError(w, r)
		return
	}

	// Get checksum result
	sum := hasher.Sum(nil)
}
