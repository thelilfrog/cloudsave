package run

import (
	"cloudsave/pkg/repository"
	"cloudsave/pkg/tools/archive"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
)

type (
	RunCmd struct {
	}
)

func (*RunCmd) Name() string     { return "run" }
func (*RunCmd) Synopsis() string { return "Check and process all the folder" }
func (*RunCmd) Usage() string {
	return `run:
  Check and process all the folder
`
}

func (p *RunCmd) SetFlags(f *flag.FlagSet) {}

func (p *RunCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	datastore, err := repository.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	for _, metadata := range datastore {
		metadataPath := filepath.Join(repository.DatastorePath(), metadata.ID)
		//todo transaction
		err := archiveIfChanged(metadata.ID, metadata.Path, filepath.Join(metadataPath, "data.tar.gz"), filepath.Join(metadataPath, ".last_run"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		if err := repository.SetVersion(metadata.ID, metadata.Version+1); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		if err := repository.SetDate(metadata.ID, time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		fmt.Println("✅", metadata.Name)
	}

	fmt.Println("done.")
	return subcommands.ExitSuccess
}

// archiveIfChanged will archive srcDir into destTarGz only if any file
// in srcDir has a modification time > the last run time stored in stateFile.
// After archiving, it updates stateFile to the current time.
func archiveIfChanged(gameID, srcDir, destTarGz, stateFile string) error {
	pg := progressbar.New(-1)
	destroyPg := func() {
		pg.Finish()
		pg.Clear()
		pg.Close()

	}
	defer destroyPg()

	pg.Describe("Scanning " + gameID + "...")

	// load last run time
	var lastRun time.Time
	data, err := os.ReadFile(stateFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to reading state file: %w", err)
	}
	if err == nil {
		lastRun, err = time.Parse(time.RFC3339, string(data))
		if err != nil {
			return fmt.Errorf("parsing state file timestamp: %w", err)
		}
	}

	// check for changes
	changed := false
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.ModTime().After(lastRun) {
			changed = true
			return io.EOF // early exit
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to scanning source directory: %w", err)
	}

	if !changed {
		pg.Finish()
		return nil
	}

	// make a backup
	pg.Describe("Backup current data...")
	if err := repository.MakeArchive(gameID); err != nil {
		return fmt.Errorf("failed to archive data: %w", err)
	}

	// create archive
	pg.Describe("Archiving new data...")
	f, err := os.Create(destTarGz)
	if err != nil {
		return fmt.Errorf("failed to creating archive file: %w", err)
	}
	defer f.Close()

	if err := archive.Tar(f, srcDir); err != nil {
		return fmt.Errorf("failed archiving files")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := os.WriteFile(stateFile, []byte(now), 0644); err != nil {
		return fmt.Errorf("updating state file: %w", err)
	}

	return nil
}
