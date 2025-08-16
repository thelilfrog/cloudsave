package main

import (
	"cloudsave/cmd/cli/commands/add"
	"cloudsave/cmd/cli/commands/apply"
	"cloudsave/cmd/cli/commands/list"
	"cloudsave/cmd/cli/commands/pull"
	"cloudsave/cmd/cli/commands/remote"
	"cloudsave/cmd/cli/commands/remove"
	"cloudsave/cmd/cli/commands/run"
	"cloudsave/cmd/cli/commands/sync"
	"cloudsave/cmd/cli/commands/version"
	"cloudsave/pkg/data"
	"cloudsave/pkg/repository"
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
)

func main() {
	roaming, err := os.UserConfigDir()
	if err != nil {
		panic("failed to get user config path: " + err.Error())
	}

	datastorepath := filepath.Join(roaming, "cloudsave", "data")
	err = os.MkdirAll(datastorepath, 0740)
	if err != nil {
		panic("cannot make the datastore:" + err.Error())
	}

	repo, err := repository.NewLazyRepository(datastorepath)
	if err != nil {
		panic("cannot make the datastore:" + err.Error())
	}

	s := data.NewService(repo)

	subcommands.Register(subcommands.HelpCommand(), "help")
	subcommands.Register(subcommands.FlagsCommand(), "help")
	subcommands.Register(subcommands.CommandsCommand(), "help")
	subcommands.Register(&version.VersionCmd{}, "help")

	subcommands.Register(&add.AddCmd{Service: s}, "management")
	subcommands.Register(&run.RunCmd{Service: s}, "management")
	subcommands.Register(&list.ListCmd{Service: s}, "management")
	subcommands.Register(&remove.RemoveCmd{Service: s}, "management")

	subcommands.Register(&apply.ListCmd{Service: s}, "restore")

	subcommands.Register(&remote.RemoteCmd{Service: s}, "remote")
	subcommands.Register(&sync.SyncCmd{Service: s}, "remote")
	subcommands.Register(&pull.PullCmd{Service: s}, "remote")

	flag.Parse()
	ctx := context.Background()

	os.Exit(int(subcommands.Execute(ctx)))
}
