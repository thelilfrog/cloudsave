package data

import (
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/repository"
	"cloudsave/pkg/tools/archive"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type (
	Service struct {
		repo repository.Repository
	}
)

func NewService(repo repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Add(name, path, remote string) (string, error) {
	gameID := repository.NewGameIdentifier(uuid.NewString())

	if err := s.repo.Mkdir(gameID); err != nil {
		return "", fmt.Errorf("failed to add game reference: %w", err)
	}

	m := repository.Metadata{
		ID:      gameID.Key(),
		Name:    name,
		Path:    path,
		Version: 1,
		Date:    time.Now(),
	}

	if err := s.repo.WriteMetadata(gameID, m); err != nil {
		return "", fmt.Errorf("failed to add game reference: %w", err)
	}

	return gameID.Key(), nil
}

func (s *Service) One(gameID string) (repository.Metadata, error) {
	id := repository.NewGameIdentifier(gameID)

	m, err := s.repo.Metadata(id)
	if err != nil {
		return repository.Metadata{}, fmt.Errorf("failed to get metadata: %w", err)
	}

	return m, nil
}

func (s *Service) Backup(gameID, backupID string) (repository.Backup, error) {
	id := repository.NewBackupIdentifier(gameID, backupID)

	if err := s.repo.Mkdir(id); err != nil {
		return repository.Backup{}, fmt.Errorf("failed to make game dir: %w", err)
	}

	return s.repo.Backup(id)
}

func (s *Service) UpdateMetadata(gameID string, m repository.Metadata) error {
	id := repository.NewGameIdentifier(gameID)

	if err := s.repo.Mkdir(id); err != nil {
		return fmt.Errorf("failed to make game dir: %w", err)
	}

	if err := s.repo.WriteMetadata(id, m); err != nil {
		return fmt.Errorf("failed to write metadate: %w", err)
	}

	return nil
}

func (s *Service) Scan(gameID string) error {
	id := repository.NewGameIdentifier(gameID)

	lastRun, err := s.repo.LastScan(id)
	if err != nil {
		return fmt.Errorf("failed to get last scan time: %w", err)
	}

	m, err := s.repo.Metadata(id)
	if err != nil {
		return fmt.Errorf("failed to get game metadata: %w", err)
	}

	if !IsDirectoryChanged(m.Path, lastRun) {
		return nil
	}

	f, err := s.repo.WriteBlob(id)
	if err != nil {
		return fmt.Errorf("failed to get datastore stream: %w", err)
	}
	if v, ok := f.(io.Closer); ok {
		defer v.Close()
	}

	if err := archive.Tar(f, m.Path); err != nil {
		return fmt.Errorf("failed to make archive: %w", err)
	}

	if err := s.repo.ResetLastScan(id); err != nil {
		return fmt.Errorf("failed to reset scan date: %w", err)
	}

	m.Date = time.Now()
	m.Version += 1

	if err := s.repo.WriteMetadata(id, m); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

func (s *Service) MakeBackup(gameID string) error {
	var id repository.Identifier = repository.NewGameIdentifier(gameID)

	src, err := s.repo.ReadBlob(id)
	if err != nil {
		return err
	}
	if v, ok := src.(io.Closer); ok {
		defer v.Close()
	}

	id = repository.NewBackupIdentifier(gameID, uuid.NewString())

	if err := s.repo.Mkdir(id); err != nil {
		return err
	}

	dst, err := s.repo.WriteBlob(id)
	if err != nil {
		return err
	}
	if v, ok := dst.(io.Closer); ok {
		defer v.Close()
	}

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

func (s *Service) AllGames() ([]repository.Metadata, error) {
	ids, err := s.repo.All()
	if err != nil {
		return nil, fmt.Errorf("failed to get the list of ids: %w", err)
	}

	var ms []repository.Metadata
	for _, id := range ids {
		m, err := s.repo.Metadata(repository.NewGameIdentifier(id))
		if err != nil {
			return nil, fmt.Errorf("failed to open metadata: %w", err)
		}
		ms = append(ms, m)
	}

	return ms, nil
}

func (s *Service) AllBackups(gameID string) ([]repository.Backup, error) {
	ids, err := s.repo.AllHist(repository.NewGameIdentifier(gameID))
	if err != nil {
		return nil, fmt.Errorf("failed to get the list of ids: %w", err)
	}

	var bs []repository.Backup
	for _, id := range ids {
		b, err := s.repo.Backup(repository.NewBackupIdentifier(gameID, id))
		if err != nil {
			return nil, fmt.Errorf("failed to open metadata: %w", err)
		}
		bs = append(bs, b)
	}

	return bs, nil
}

func (l Service) PullArchive(gameID, backupID string, cli *client.Client) error {
	if len(backupID) > 0 {
		path := l.repo.DataPath(repository.NewBackupIdentifier(gameID, backupID))
		return cli.PullBackup(gameID, backupID, filepath.Join(path, "data.tar.gz"))
	}

	path := l.repo.DataPath(repository.NewGameIdentifier(gameID))
	return cli.Pull(gameID, filepath.Join(path, "data.tar.gz"))
}

func (l Service) PushArchive(gameID, backupID string, cli *client.Client) error {
	m, err := l.repo.Metadata(repository.NewGameIdentifier(gameID))
	if err != nil {
		return err
	}

	if len(backupID) > 0 {
		path := l.repo.DataPath(repository.NewBackupIdentifier(gameID, backupID))
		return cli.PushSave(filepath.Join(path, "data.taz.gz"), m)
	}

	path := l.repo.DataPath(repository.NewGameIdentifier(gameID))
	return cli.PushSave(filepath.Join(path, "data.tar.gz"), m)
}

func (l Service) PullCurrent(id, path string, cli *client.Client) error {
	gameID := repository.NewGameIdentifier(id)
	if err := l.repo.Mkdir(gameID); err != nil {
		return err
	}

	m, err := cli.Metadata(id)
	if err != nil {
		return fmt.Errorf("failed to get metadata from the server: %w", err)
	}

	if err := l.repo.WriteMetadata(gameID, m); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	archivePath := filepath.Join(l.repo.DataPath(gameID), "data.tar.gz")

	if err := cli.Pull(id, archivePath); err != nil {
		return fmt.Errorf("failed to pull from the server: %w", err)
	}

	f, err := l.repo.ReadBlob(gameID)
	if err != nil {
		return fmt.Errorf("failed to open blob from local repository: %w", err)
	}

	if err := os.MkdirAll(path, 0740); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := archive.Untar(f, path); err != nil {
		return fmt.Errorf("failed to untar archive: %w", err)
	}

	if err := l.repo.ResetLastScan(gameID); err != nil {
		return fmt.Errorf("failed to create .last_run file: %w", err)
	}

	return nil
}

func (l Service) PullBackup(gameID, backupID string, cli *client.Client) error {
	id := repository.NewBackupIdentifier(gameID, backupID)

	archivePath := filepath.Join(l.repo.DataPath(id), "data.tar.gz")

	if err := cli.PullBackup(gameID, backupID, archivePath); err != nil {
		return fmt.Errorf("failed to pull backup: %w", err)
	}

	return nil
}

func (l Service) RemoveGame(gameID string) error {
	return l.repo.Remove(repository.NewGameIdentifier(gameID))
}

func (l Service) SetVersion(gameID string, value int) error {
	id := repository.NewGameIdentifier(gameID)

	m, err := l.repo.Metadata(id)
	if err != nil {
		return fmt.Errorf("failed to get metadata from the server: %w", err)
	}

	m.Version = value

	if err := l.repo.WriteMetadata(id, m); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func IsDirectoryChanged(path string, lastRun time.Time) bool {
	changed := false
	_ = filepath.Walk(path, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.ModTime().After(lastRun) {
			changed = true
			return io.EOF // early exit
		}
		return nil
	})
	return changed
}

func (l Service) Copy(id string, src io.Reader) error {
	dst, err := l.repo.WriteBlob(repository.NewGameIdentifier(id))
	if err != nil {
		return err
	}
	if v, ok := dst.(io.Closer); ok {
		defer v.Close()
	}

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

func (l Service) CopyBackup(gameID, backupID string, src io.Reader) error {
	id := repository.NewBackupIdentifier(gameID, backupID)

	if err := l.repo.Mkdir(id); err != nil {
		return err
	}

	dst, err := l.repo.WriteBlob(id)
	if err != nil {
		return err
	}
	if v, ok := dst.(io.Closer); ok {
		defer v.Close()
	}

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}
