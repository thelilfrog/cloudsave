package main

import (
	"cloudsave/cmd/cli/commands/add"
	"cloudsave/cmd/cli/commands/list"
	"cloudsave/cmd/cli/commands/remote"
	"cloudsave/cmd/cli/commands/remove"
	"cloudsave/cmd/cli/commands/run"
	"cloudsave/cmd/cli/commands/sync"
	"cloudsave/cmd/cli/commands/version"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "help")
	subcommands.Register(subcommands.FlagsCommand(), "help")
	subcommands.Register(subcommands.CommandsCommand(), "help")
	subcommands.Register(&version.VersionCmd{}, "help")

	subcommands.Register(&add.AddCmd{}, "management")
	subcommands.Register(&run.RunCmd{}, "management")
	subcommands.Register(&list.ListCmd{}, "management")
	subcommands.Register(&remove.RemoveCmd{}, "management")

	subcommands.Register(&remote.RemoteCmd{}, "remote")
	subcommands.Register(&sync.SyncCmd{}, "remote")

	flag.Parse()
	ctx := context.Background()

	exitCode := subcommands.Execute(ctx)
	fmt.Println()

	os.Exit(int(exitCode))
}
