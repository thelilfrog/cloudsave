package remote

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func ConnectWithKey(host, user string) (*sftp.Client, error) {
	authMethods := loadSshKeys()

	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: authMethods,
	}

	return connect(host, config)
}

func ConnectWithPassword(host, user string) (*sftp.Client, error) {
	fmt.Printf("%s@%s's password:", user, host)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}

	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(bytePassword)),
		},
	}

	return connect(host, config)
}

func connect(host string, config *ssh.ClientConfig) (*sftp.Client, error) {
	// Connect to the SSH server
	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ssh server: %w", err)
	}
	defer conn.Close()

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ssh server: %w", err)
	}
	return sftpClient, nil
}

func loadSshKeys() []ssh.AuthMethod {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	var auths []ssh.AuthMethod
	entries, err := os.ReadDir(filepath.Join(dirname, ".ssh"))
	if err != nil {
		return auths
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		keyPath := filepath.Join(dirname, name)
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			fmt.Printf("%s's passphrase:", entry.Name())
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				continue
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, bytePassword)
			if err != nil {
				continue
			}
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}

	return auths
}
