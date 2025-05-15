package sync

import (
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/tools/prompt/credentials"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	remotes, err := remote.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load datastore:", err)
		return subcommands.ExitFailure
	}

	username, password, err := credentials.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to read std output:", err)
		return subcommands.ExitFailure
	}

	for _, r := range remotes {
		if !ping(r.URL, username, password) {
			slog.Warn("remote is unavailable", "url", r.URL)
			continue
		}

		client := client.New(r.URL, username, password)

		hlocal, err := game.Hash(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		hremote, _ := client.Hash(r.GameID)

		vlocal, err := game.Version(r.GameID)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		vremote, _ := client.Version(r.GameID)

		if hlocal == hremote {
			fmt.Println("already up-to-date")
			continue
		}

		if vremote == 0 {
			fmt.Println("push")
			continue
		}

		if vlocal > vremote {
			fmt.Println("push")
			continue
		}

		if vlocal < vremote {
			fmt.Println("pull")
			continue
		}

		if vlocal == vremote {
			fmt.Println("conflict")
			continue
		}
	}

	return subcommands.ExitSuccess
}

func ping(remote, username, password string) bool {
	cli := http.Client{}

	hburl, err := url.JoinPath(remote, "heartbeat")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	req, err := http.NewRequest("GET", hburl, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	req.SetBasicAuth(username, password)

	res, err := cli.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	if res.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "cannot connect to remote: server return code", res.StatusCode)
		return false
	}

	return true
}

