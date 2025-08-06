package list

import (
	"cloudsave/cmd/cli/tools/prompt/credentials"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/repository"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type (
	ListCmd struct {
		remote bool
		backup bool
	}
)

func (*ListCmd) Name() string     { return "list" }
func (*ListCmd) Synopsis() string { return "list all game registered" }
func (*ListCmd) Usage() string {
	return `Usage: cloudsave list [-include-backup] [-a]

List all game registered

Options:
`
}

func (p *ListCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.remote, "a", false, "list all including remote data")
	f.BoolVar(&p.backup, "include-backup", false, "include backup uuids in the output")
}

func (p *ListCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.remote {
		if f.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "error: missing remote url")
			return subcommands.ExitUsageError
		}

		username, password, err := credentials.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read std output: %s", err)
			return subcommands.ExitFailure
		}

		if err := remote(f.Arg(0), username, password, p.backup); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}
	if err := local(p.backup); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func local(includeBackup bool) error {
	games, err := repository.All()
	if err != nil {
		return fmt.Errorf("failed to load datastore: %w", err)
	}

	for _, g := range games {
		fmt.Println("ID:", g.ID)
		fmt.Println("Name:", g.Name)
		fmt.Println("Last Version:", g.Date, "( Version Number", g.Version, ")")
		if includeBackup {
			bk, err := repository.Archives(g.ID)
			if err != nil {
				return fmt.Errorf("failed to list backup files: %w", err)
			}
			if len(bk) > 0 {
				fmt.Println("Backup:")
				for _, b := range bk {
					fmt.Printf("   - %s (%s)\n", b.UUID, b.CreatedAt)
				}
			}
		}
		fmt.Println("---")
	}

	return nil
}

func remote(url, username, password string, includeBackup bool) error {
	cli := client.New(url, username, password)

	if err := cli.Ping(); err != nil {
		return fmt.Errorf("failed to connect to the remote: %w", err)
	}

	games, err := cli.All()
	if err != nil {
		return fmt.Errorf("failed to load games from remote: %w", err)
	}

	fmt.Println()
	fmt.Println("Remote:", url)
	fmt.Println("---")
	for _, g := range games {
		fmt.Println("ID:", g.ID)
		fmt.Println("Name:", g.Name)
		fmt.Println("Last Version:", g.Date, "( Version Number", g.Version, ")")
		if includeBackup {
			bk, err := cli.ListArchives(g.ID)
			if err != nil {
				return fmt.Errorf("failed to list backup files: %w", err)
			}
			if len(bk) > 0 {
				fmt.Println("Backup:")
				for _, uuid := range bk {
					b, err := cli.ArchiveInfo(g.ID, uuid)
					if err != nil {
						return fmt.Errorf("failed to list backup files: %w", err)
					}
					fmt.Printf("   - %s (%s)\n", b.UUID, b.CreatedAt)
				}
			}
		}
		fmt.Println("---")
	}

	return nil
}
