package main

import (
	"cloudsave/cmd/cli/commands/add"
	"cloudsave/cmd/cli/commands/list"
	"cloudsave/cmd/cli/commands/remote"
	"cloudsave/cmd/cli/commands/remove"
	"cloudsave/cmd/cli/commands/run"
	"cloudsave/cmd/cli/commands/sync"
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "help")
	subcommands.Register(subcommands.FlagsCommand(), "help")
	subcommands.Register(subcommands.CommandsCommand(), "help")

	subcommands.Register(&add.AddCmd{}, "management")
	subcommands.Register(&run.RunCmd{}, "management")
	subcommands.Register(&list.ListCmd{}, "management")
	subcommands.Register(&remove.RemoveCmd{}, "management")

	subcommands.Register(&remote.RemoteCmd{}, "remote")
	subcommands.Register(&sync.SyncCmd{}, "remote")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
