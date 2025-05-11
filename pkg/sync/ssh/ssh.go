package ssh

import (
	"cloudsave/pkg/remote"
	"fmt"
	"log"
	"os/user"
)

type (
	SFTPSyncer struct {
	}
)

func (SFTPSyncer) Sync(r remote.Remote) error {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}
	cli, err := remote.ConnectWithKey(r.URL, currentUser.Username)
	if err != nil {
		cli, err = remote.ConnectWithPassword(r.URL, currentUser.Username)
		if err != nil {
			return fmt.Errorf("failed to connect to host: %w", err)
		}
	}
	defer cli.Close()

	

	return nil
}
