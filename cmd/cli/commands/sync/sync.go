package sync

import (
	"cloudsave/cmd/cli/tools/prompt"
	"cloudsave/cmd/cli/tools/prompt/credentials"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/repository"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
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
	games, err := repository.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	remoteCred := make(map[string]map[string]string)
	for _, g := range games {
		r, err := remote.One(g.ID)
		if err != nil {
			if errors.Is(err, remote.ErrNoRemote) {
				continue
			}
			fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
			return subcommands.ExitFailure
		}
		cli, err := connect(remoteCred, r)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to connect to the remote:", err)
			return subcommands.ExitFailure
		}

		pg := progressbar.New(-1)
		destroyPg := func() {
			pg.Finish()
			pg.Clear()
			pg.Close()

		}

		pg.Describe(fmt.Sprintf("[%s] Checking status...", g.Name))
		exists, err := cli.Exists(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		if !exists {
			pg.Describe(fmt.Sprintf("[%s] Pushing data...", g.Name))
			if err := push(g, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			pg.Describe(fmt.Sprintf("[%s] Pushing backup...", g.Name))
			if err := pushBackup(g, cli); err != nil {
				destroyPg()
				slog.Warn("failed to push backup files", "err", err)
			}
			continue
		}

		pg.Describe(fmt.Sprintf("[%s] Fetching metadata...", g.Name))
		hlocal, err := repository.Hash(r.GameID)
		if err != nil {
			destroyPg()
			slog.Error(err.Error())
			continue
		}

		hremote, err := cli.Hash(r.GameID)
		if err != nil {
			destroyPg()
			fmt.Fprintln(os.Stderr, "error: failed to get the file hash from the remote:", err)
			continue
		}

		vlocal, err := repository.Version(r.GameID)
		if err != nil {
			destroyPg()
			slog.Error(err.Error())
			continue
		}

		remoteMetadata, err := cli.Metadata(r.GameID)
		if err != nil {
			destroyPg()
			fmt.Fprintln(os.Stderr, "error: failed to get the game metadata from the remote:", err)
			continue
		}

		pg.Describe(fmt.Sprintf("[%s] Pulling backup...", g.Name))
		if err := pullBackup(g, cli); err != nil {
			slog.Warn("failed to pull backup files", "err", err)
		}

		pg.Describe(fmt.Sprintf("[%s] Pushing backup...", g.Name))
		if err := pushBackup(g, cli); err != nil {
			slog.Warn("failed to push backup files", "err", err)
		}

		if hlocal == hremote {
			destroyPg()
			if vlocal != remoteMetadata.Version {
				slog.Debug("version is not the same, but the hash is equal. Updating local database")
				if err := repository.SetVersion(r.GameID, remoteMetadata.Version); err != nil {
					fmt.Fprintln(os.Stderr, "error: failed to synchronize version number:", err)
					continue
				}
			}
			fmt.Println("already up-to-date")
			continue
		}

		if vlocal > remoteMetadata.Version {
			pg.Describe(fmt.Sprintf("[%s] Pushing data...", g.Name))
			if err := push(g, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			destroyPg()
			continue
		}

		if vlocal < remoteMetadata.Version {
			destroyPg()
			if err := pull(r.GameID, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			if err := repository.SetVersion(r.GameID, remoteMetadata.Version); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "error: failed to synchronize version number:", err)
				continue
			}
			if err := repository.SetDate(r.GameID, remoteMetadata.Date); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "error: failed to synchronize date:", err)
				continue
			}
			continue
		}

		destroyPg()

		if vlocal == remoteMetadata.Version {
			if err := conflict(r.GameID, g, remoteMetadata, cli); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to resolve conflict:", err)
				continue
			}
			continue
		}
	}

	fmt.Println("done.")
	return subcommands.ExitSuccess
}

func conflict(gameID string, m, remoteMetadata repository.Metadata, cli *client.Client) error {
	g, err := repository.One(gameID)
	if err != nil {
		slog.Warn("a conflict was found but the game is not found in the database")
		slog.Debug("debug info", "gameID", gameID)
		return nil
	}
	fmt.Println()
	fmt.Println("--- /!\\ CONFLICT ---")
	fmt.Println(g.Name, "(", g.Path, ")")
	fmt.Println("----")
	fmt.Println("Your version:", g.Date.Format(time.RFC1123))
	fmt.Println("Their version:", remoteMetadata.Date.Format(time.RFC1123))
	fmt.Println()

	res := prompt.Conflict()

	switch res {
	case prompt.My:
		{
			if err := push(m, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}

	case prompt.Their:
		{
			if err := pull(gameID, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
			if err := repository.SetVersion(gameID, remoteMetadata.Version); err != nil {
				return fmt.Errorf("failed to synchronize version number: %w", err)
			}
			if err := repository.SetDate(gameID, remoteMetadata.Date); err != nil {
				return fmt.Errorf("failed to synchronize date: %w", err)
			}
		}
	}
	return nil
}

func push(m repository.Metadata, cli *client.Client) error {
	archivePath := filepath.Join(repository.DatastorePath(), m.ID, "data.tar.gz")

	return cli.PushSave(archivePath, m)
}

func pushBackup(m repository.Metadata, cli *client.Client) error {
	bs, err := repository.Archives(m.ID)
	if err != nil {
		return err
	}

	for _, b := range bs {
		binfo, err := cli.ArchiveInfo(m.ID, b.UUID)
		if err != nil {
			if !errors.Is(err, client.ErrNotFound) {
				return fmt.Errorf("failed to get remote information about the backup file: %w", err)
			}
		}

		if binfo.MD5 != b.MD5 {
			if err := cli.PushBackup(b, m); err != nil {
				return fmt.Errorf("failed to push backup: %w", err)
			}
		}

	}
	return nil
}

func pullBackup(m repository.Metadata, cli *client.Client) error {
	bs, err := cli.ListArchives(m.ID)
	if err != nil {
		return err
	}

	for _, uuid := range bs {
		rinfo, err := cli.ArchiveInfo(m.ID, uuid)
		if err != nil {
			return err
		}

		linfo, err := repository.Archive(m.ID, uuid)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}

		path := filepath.Join(repository.DatastorePath(), m.ID, "hist", uuid)
		if err := os.MkdirAll(path, 0740); err != nil {
			return err
		}

		if rinfo.MD5 != linfo.MD5 {
			if err := cli.PullBackup(m.ID, uuid, filepath.Join(path, "data.tar.gz")); err != nil {
				return err
			}
		}
	}
	return nil
}

func pull(gameID string, cli *client.Client) error {
	archivePath := filepath.Join(repository.DatastorePath(), gameID, "data.tar.gz")

	return cli.Pull(gameID, archivePath)
}

func connect(remoteCred map[string]map[string]string, r remote.Remote) (*client.Client, error) {
	var cli *client.Client

	if v, ok := remoteCred[r.URL]; ok {
		cli = client.New(r.URL, v["username"], v["password"])
		return cli, nil
	}

	username, password, err := credentials.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read std output: %w", err)
	}

	cli = client.New(r.URL, username, password)

	if err := cli.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the remote: %w", err)
	}

	c := make(map[string]string)
	c["username"] = username
	c["password"] = password
	remoteCred[r.URL] = c

	return cli, nil
}
