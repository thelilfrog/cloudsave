package run

import (
	"archive/tar"
	"cloudsave/pkg/game"
	"compress/gzip"
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
	datastore, err := game.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	pg := progressbar.New(len(datastore))
	defer pg.Close()

	for _, metadata := range datastore {
		pg.Describe("Scanning " + metadata.Name + "...")
		metadataPath := filepath.Join(game.DatastorePath(), metadata.ID)
		//todo transaction
		err := archiveIfChanged(metadata.ID, metadata.Path, filepath.Join(metadataPath, "data.tar.gz"), filepath.Join(metadataPath, ".last_run"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		if err := game.SetVersion(metadata.ID, metadata.Version+1); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		if err := game.SetDate(metadata.ID, time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot process the data of %s: %s\n", metadata.ID, err)
			return subcommands.ExitFailure
		}
		pg.Add(1)
	}

	pg.Finish()

	return subcommands.ExitSuccess
}

// archiveIfChanged will archive srcDir into destTarGz only if any file
// in srcDir has a modification time > the last run time stored in stateFile.
// After archiving, it updates stateFile to the current time.
func archiveIfChanged(id, srcDir, destTarGz, stateFile string) error {
	// 1) Load last run time
	var lastRun time.Time
	data, err := os.ReadFile(stateFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading state file: %w", err)
	}
	if err == nil {
		lastRun, err = time.Parse(time.RFC3339, string(data))
		if err != nil {
			return fmt.Errorf("parsing state file timestamp: %w", err)
		}
	}

	// 2) Check for changes
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
		return fmt.Errorf("scanning source directory: %w", err)
	}

	if !changed {
		return nil
	}

	// 3) Create tar.gz
	f, err := os.Create(destTarGz)
	if err != nil {
		return fmt.Errorf("creating archive file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Walk again to add files
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Create tar header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}
		// Preserve directory structure relative to srcDir
		relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("writing tar entries: %w", err)
	}

	// 4) Update state file
	now := time.Now().UTC().Format(time.RFC3339)
	if err := os.WriteFile(stateFile, []byte(now), 0644); err != nil {
		return fmt.Errorf("updating state file: %w", err)
	}

	return nil
}
