package sync

import (
	"cloudsave/cmd/cli/tools/prompt"
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/tools/prompt/credentials"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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
	games, err := game.All()
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

		exists, err := cli.Exists(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		if !exists {
			if err := push(r.GameID, g, cli); err != nil {
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			continue
		}

		hlocal, err := game.Hash(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		hremote, err := cli.Hash(r.GameID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to get the file hash from the remote:", err)
			continue
		}

		vlocal, err := game.Version(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		remoteMetadata, err := cli.Metadata(r.GameID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to get the game metadata from the remote:", err)
			continue
		}

		if hlocal == hremote {
			if vlocal != remoteMetadata.Version {
				slog.Debug("version is not the same, but the hash is equal. Updating local database")
				if err := game.SetVersion(r.GameID, remoteMetadata.Version); err != nil {
					fmt.Fprintln(os.Stderr, "error: failed to synchronize version number:", err)
					continue
				}
			}
			fmt.Println("already up-to-date")
			continue
		}

		if vlocal > remoteMetadata.Version {
			if err := push(r.GameID, g, cli); err != nil {
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			continue
		}

		if vlocal < remoteMetadata.Version {
			if err := pull(r.GameID, cli); err != nil {
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			if err := game.SetVersion(r.GameID, remoteMetadata.Version); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to synchronize version number:", err)
				continue
			}
			if err := game.SetDate(r.GameID, remoteMetadata.Date); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to synchronize date:", err)
				continue
			}
			continue
		}

		if vlocal == remoteMetadata.Version {
			if err := conflict(r.GameID, g, remoteMetadata, cli); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to resolve conflict:", err)
				continue
			}
			continue
		}
	}

	return subcommands.ExitSuccess
}

func conflict(gameID string, m, remoteMetadata game.Metadata, cli *client.Client) error {
	g, err := game.One(gameID)
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
			if err := push(gameID, m, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}

	case prompt.Their:
		{
			if err := pull(gameID, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
			if err := game.SetVersion(gameID, remoteMetadata.Version); err != nil {
				return fmt.Errorf("failed to synchronize version number: %w", err)
			}
			if err := game.SetDate(gameID, remoteMetadata.Date); err != nil {
				return fmt.Errorf("failed to synchronize date: %w", err)
			}
		}
	}
	return nil
}

func push(gameID string, m game.Metadata, cli *client.Client) error {
	archivePath := filepath.Join(game.DatastorePath(), gameID, "data.tar.gz")

	return cli.Push(gameID, archivePath, m)
}

func pull(gameID string, cli *client.Client) error {
	archivePath := filepath.Join(game.DatastorePath(), gameID, "data.tar.gz")

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
