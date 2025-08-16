package sync

import (
	"cloudsave/cmd/cli/tools/prompt"
	"cloudsave/cmd/cli/tools/prompt/credentials"
	"cloudsave/pkg/data"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/repository"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
)

type (
	SyncCmd struct {
		Service *data.Service
	}
)

func (*SyncCmd) Name() string     { return "sync" }
func (*SyncCmd) Synopsis() string { return "list all game registered" }
func (*SyncCmd) Usage() string {
	return `Usage: cloudsave sync

Synchronize the archives with the server defined for each game.
`
}

func (p *SyncCmd) SetFlags(f *flag.FlagSet) {
}

func (p *SyncCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	games, err := p.Service.AllGames()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	remoteCred := make(map[string]map[string]string)
	for _, g := range games {
		r, err := remote.One(g.ID)
		if err != nil {
			if errors.Is(err, remote.ErrNoRemote) {
				fmt.Println(g.Name + ": no remote configured")
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
			if err := p.push(g, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			pg.Describe(fmt.Sprintf("[%s] Pushing backup...", g.Name))
			if err := p.pushBackup(g, cli); err != nil {
				destroyPg()
				slog.Warn("failed to push backup files", "err", err)
			}
			fmt.Println(g.Name + ": pushed")
			continue
		}

		pg.Describe(fmt.Sprintf("[%s] Fetching metadata...", g.Name))

		hremote, err := cli.Hash(r.GameID)
		if err != nil {
			destroyPg()
			fmt.Fprintln(os.Stderr, "error: failed to get the file hash from the remote:", err)
			continue
		}

		remoteMetadata, err := cli.Metadata(r.GameID)
		if err != nil {
			destroyPg()
			fmt.Fprintln(os.Stderr, "error: failed to get the game metadata from the remote:", err)
			continue
		}

		pg.Describe(fmt.Sprintf("[%s] Pulling backup...", g.Name))
		if err := p.pullBackup(g, cli); err != nil {
			slog.Warn("failed to pull backup files", "err", err)
		}

		pg.Describe(fmt.Sprintf("[%s] Pushing backup...", g.Name))
		if err := p.pushBackup(g, cli); err != nil {
			slog.Warn("failed to push backup files", "err", err)
		}

		if g.MD5 == hremote {
			destroyPg()
			if g.Version != remoteMetadata.Version {
				slog.Debug("version is not the same, but the hash is equal. Updating local database")
				if err := p.Service.SetVersion(r.GameID, remoteMetadata.Version); err != nil {
					fmt.Fprintln(os.Stderr, "error: failed to synchronize version number:", err)
					continue
				}
			}
			fmt.Println(g.Name + ": already up-to-date")
			continue
		}

		if g.Version > remoteMetadata.Version {
			pg.Describe(fmt.Sprintf("[%s] Pushing data...", g.Name))
			if err := p.push(g, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			destroyPg()
			fmt.Println(g.Name + ": pushed")
			continue
		}

		if g.Version < remoteMetadata.Version {
			destroyPg()
			if err := p.pull(r.GameID, cli); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}

			g.Version = remoteMetadata.Version
			g.Date = remoteMetadata.Date

			if err := p.Service.UpdateMetadata(g.ID, g); err != nil {
				destroyPg()
				fmt.Fprintln(os.Stderr, "failed to push:", err)
				return subcommands.ExitFailure
			}
			fmt.Println(g.Name + ": pulled")
			continue
		}

		destroyPg()

		if g.Version == remoteMetadata.Version {
			if err := p.conflict(r.GameID, g, remoteMetadata, cli); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to resolve conflict:", err)
				continue
			}
			continue
		}
	}

	fmt.Println("done.")
	return subcommands.ExitSuccess
}

func (p *SyncCmd) conflict(gameID string, m, remoteMetadata repository.Metadata, cli *client.Client) error {
	g, err := p.Service.One(gameID)
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
			if err := p.push(m, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}

	case prompt.Their:
		{
			if err := p.pull(gameID, cli); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
			g.Version = remoteMetadata.Version
			g.Date = remoteMetadata.Date

			if err := p.Service.UpdateMetadata(g.ID, g); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}
	}
	return nil
}

func (p *SyncCmd) push(m repository.Metadata, cli *client.Client) error {
	return p.Service.PushArchive(m.ID, "", cli)
}

func (p *SyncCmd) pushBackup(m repository.Metadata, cli *client.Client) error {
	bs, err := p.Service.AllBackups(m.ID)
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

func (p *SyncCmd) pullBackup(m repository.Metadata, cli *client.Client) error {
	bs, err := cli.ListArchives(m.ID)
	if err != nil {
		return err
	}

	for _, uuid := range bs {
		rinfo, err := cli.ArchiveInfo(m.ID, uuid)
		if err != nil {
			return err
		}

		linfo, err := p.Service.Backup(m.ID, uuid)
		if err != nil {
			return err
		}

		if linfo.MD5 != rinfo.MD5 {
			if err := p.Service.PullBackup(m.ID, uuid, cli); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *SyncCmd) pull(gameID string, cli *client.Client) error {
	return p.Service.PullArchive(gameID, "", cli)
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
