package sync

import (
	"cloudsave/cmd/cli/tools/prompt"
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/tools/prompt/credentials"
	"context"
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
	remotes, err := remote.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	if len(remotes) == 0 {
		fmt.Println("nothing to do: no remote found")
		return subcommands.ExitSuccess
	}

	username, password, err := credentials.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to read std output:", err)
		return subcommands.ExitFailure
	}

	for _, r := range remotes {
		m, err := game.One(r.GameID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot get metadata for this game: %w", err)
			return subcommands.ExitFailure
		}

		client := client.New(r.URL, username, password)

		if !client.Ping() {
			slog.Warn("remote is unavailable", "url", r.URL)
			continue
		}

		exists, err := client.Exists(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		if !exists {
			if err := push(r.GameID, m, client); err != nil {
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

		hremote, err := client.Hash(r.GameID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to get the file hash from the remote:", err)
			continue
		}

		vlocal, err := game.Version(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		remoteMetadata, err := client.Metadata(r.GameID)
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
			if err := push(r.GameID, m, client); err != nil {
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			continue
		}

		if vlocal < remoteMetadata.Version {
			if err := pull(r.GameID, client); err != nil {
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
			g, err := game.One(r.GameID)
			if err != nil {
				slog.Warn("a conflict was found but the game is not found in the database")
				slog.Debug("debug info", "gameID", r.GameID)
				continue
			}
			fmt.Println()
			fmt.Println("--- /!\\ CONFLICT ---")
			fmt.Println("----")
			fmt.Println(g.Name, "(", g.Path, ")")
			fmt.Println("----")
			fmt.Println("Your version:", g.Date.Format(time.RFC1123))
			fmt.Println("Their version:", remoteMetadata.Date.Format(time.RFC1123))
			fmt.Println()

			res := prompt.Conflict()

			switch res {
			case prompt.My:
				{
					if err := push(r.GameID, m, client); err != nil {
						fmt.Fprintln(os.Stderr, "failed to push:", err)
						return subcommands.ExitFailure
					}
				}

			case prompt.Their:
				{
					if err := pull(r.GameID, client); err != nil {
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
				}
			}
			continue
		}

	}

	return subcommands.ExitSuccess
}

func push(gameID string, m game.Metadata, cli *client.Client) error {
	archivePath := filepath.Join(game.DatastorePath(), gameID, "data.tar.gz")

	return cli.Push(gameID, archivePath, m)
}

func pull(gameID string, cli *client.Client) error {
	archivePath := filepath.Join(game.DatastorePath(), gameID, "data.tar.gz")

	return cli.Pull(gameID, archivePath)
}
