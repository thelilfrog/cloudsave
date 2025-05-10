package main

import (
	"cloudsave/cmd/cli/commands/add"
	"cloudsave/cmd/cli/commands/run"
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "help")
	subcommands.Register(subcommands.FlagsCommand(), "help")
	subcommands.Register(subcommands.CommandsCommand(), "help")

	subcommands.Register(add.AddCmd{}, "management")
	subcommands.Register(run.RunCmd{}, "management")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
