package repository

import (
	"cloudsave/pkg/tools/hash"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type (
	Metadata struct {
		ID      string    `json:"id"`
		Name    string    `json:"name"`
		Path    string    `json:"path"`
		Version int       `json:"version"`
		Date    time.Time `json:"date"`
		MD5     string    `json:"md5,omitempty"`
	}

	Remote struct {
		URL    string `json:"url"`
		GameID string `json:"-"`
	}

	Backup struct {
		CreatedAt   time.Time `json:"created_at"`
		MD5         string    `json:"md5"`
		UUID        string    `json:"uuid"`
		ArchivePath string    `json:"-"`
	}

	Data struct {
		Metadata Metadata
		Remote   *Remote
		DataPath string
		Backup   map[string]Backup
	}

	GameIdentifier struct {
		gameID string
	}

	BackupIdentifier struct {
		gameID   string
		backupID string
	}

	Identifier interface {
		Key() string
	}

	LazyRepository struct {
		dataRoot string
	}

	EagerRepository struct {
		Repository

		mu   sync.RWMutex
		data map[string]Data
	}

	Repository interface {
		Mkdir(id Identifier) error

		All() ([]string, error)
		AllHist(gameID GameIdentifier) ([]string, error)

		WriteBlob(ID Identifier) (io.Writer, error)
		WriteMetadata(gameID GameIdentifier, m Metadata) error

		Metadata(gameID GameIdentifier) (Metadata, error)
		LastScan(gameID GameIdentifier) (time.Time, error)
		ReadBlob(gameID Identifier) (io.ReadSeekCloser, error)
		Backup(id BackupIdentifier) (Backup, error)
		Remote(id GameIdentifier) (*Remote, error)

		SetRemote(gameID GameIdentifier, url string) error
		ResetLastScan(id GameIdentifier) error

		DataPath(id Identifier) string

		Remove(gameID GameIdentifier) error
	}
)

var (
	ErrNotFound error = errors.New("not found")
)

func NewGameIdentifier(gameID string) GameIdentifier {
	return GameIdentifier{
		gameID: gameID,
	}
}
func (bi GameIdentifier) Key() string {
	return bi.gameID
}

func NewBackupIdentifier(gameID, backupID string) BackupIdentifier {
	return BackupIdentifier{
		gameID:   gameID,
		backupID: backupID,
	}
}

func (bi BackupIdentifier) Key() string {
	return bi.gameID + ":" + bi.backupID
}

func NewLazyRepository(dataRootPath string) (*LazyRepository, error) {
	if m, err := os.Stat(dataRootPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dataRootPath, 0740); err != nil {
				return nil, fmt.Errorf("failed to make the directory: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open datastore: %w", err)
		}
	} else {
		if !m.IsDir() {
			return nil, fmt.Errorf("failed to open datastore: not a directory")
		}
	}

	return &LazyRepository{
		dataRoot: dataRootPath,
	}, nil
}

func (l *LazyRepository) Mkdir(id Identifier) error {
	path := l.DataPath(id)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		slog.Debug("making directory", "path", path, "id", id, "perm", "0740")
		return os.MkdirAll(path, 0740)
	}
	return nil
}

func (l *LazyRepository) All() ([]string, error) {
	slog.Debug("loading all current data...")
	dir, err := os.ReadDir(l.dataRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}

	var res []string
	for _, d := range dir {
		res = append(res, d.Name())
	}

	return res, nil
}

func (l *LazyRepository) AllHist(id GameIdentifier) ([]string, error) {
	path := l.DataPath(id)

	slog.Debug("loading hist data...", "id", id)
	dir, err := os.ReadDir(filepath.Join(path, "hist"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}

	var res []string
	for _, d := range dir {
		res = append(res, d.Name())
	}

	return res, nil
}

func (l *LazyRepository) WriteBlob(ID Identifier) (io.Writer, error) {
	path := l.DataPath(ID)

	slog.Debug("loading write buffer...", "id", ID)
	dst, err := os.OpenFile(filepath.Join(path, "data.tar.gz"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0740)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination file: %w", err)
	}

	return dst, nil
}

func (l *LazyRepository) WriteMetadata(id GameIdentifier, m Metadata) error {
	m.MD5 = ""
	path := l.DataPath(id)

	slog.Debug("writing metadata", "id", id, "metadata", m)
	dst, err := os.OpenFile(filepath.Join(path, "metadata.json"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer dst.Close()

	e := json.NewEncoder(dst)
	if err := e.Encode(m); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	return nil
}

func (l *LazyRepository) Metadata(id GameIdentifier) (Metadata, error) {
	path := l.DataPath(id)

	slog.Debug("loading metadata", "id", id)
	src, err := os.OpenFile(filepath.Join(path, "metadata.json"), os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, ErrNotFound
		}
		return Metadata{}, fmt.Errorf("corrupted datastore: failed to open metadata: %w", err)
	}

	var m Metadata
	d := json.NewDecoder(src)
	if err := d.Decode(&m); err != nil {
		return Metadata{}, fmt.Errorf("corrupted datastore: failed to parse metadata: %w", err)
	}

	if _, err := os.Stat(filepath.Join(path, "data.tar.gz")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return Metadata{}, fmt.Errorf("failed to open archive: %w", err)
	}

	slog.Debug("loading md5 hash", "id", id)
	m.MD5, err = hash.FileMD5(filepath.Join(path, "data.tar.gz"))
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to calculate md5: %w", err)
	}

	return m, nil
}

func (l *LazyRepository) Backup(id BackupIdentifier) (Backup, error) {
	path := l.DataPath(id)

	slog.Debug("loading hist metadata", "id", id)
	fs, err := os.Stat(filepath.Join(path, "data.tar.gz"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Backup{}, ErrNotFound
		}
		return Backup{}, fmt.Errorf("corrupted datastore: failed to open metadata: %w", err)
	}

	slog.Debug("loading md5 hash", "id", id)
	h, err := hash.FileMD5(filepath.Join(path, "data.tar.gz"))
	if err != nil {
		return Backup{}, fmt.Errorf("corrupted datastore: failed to open metadata: %w", err)
	}

	return Backup{
		CreatedAt:   fs.ModTime(),
		MD5:         h,
		UUID:        id.backupID,
		ArchivePath: filepath.Join(path, "data.tar.gz"),
	}, nil
}

func (l *LazyRepository) LastScan(id GameIdentifier) (time.Time, error) {
	path := l.DataPath(id)

	data, err := os.ReadFile(filepath.Join(path, ".last_run"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to reading state file: %w", err)
	}

	lastRun, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing state file timestamp: %w", err)
	}

	return lastRun, nil
}

func (l *LazyRepository) ResetLastScan(id GameIdentifier) error {
	path := l.DataPath(id)

	slog.Debug("resetting last scan datetime for", "id", id)
	f, err := os.OpenFile(filepath.Join(path, ".last_run"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	data := time.Now().Format(time.RFC3339)

	if _, err := f.WriteString(data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (l *LazyRepository) ReadBlob(id Identifier) (io.ReadSeekCloser, error) {
	path := l.DataPath(id)

	slog.Debug("loading read buffer...", "id", id)
	dst, err := os.OpenFile(filepath.Join(path, "data.tar.gz"), os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob: %w", err)
	}

	return dst, nil
}

func (l *LazyRepository) SetRemote(id GameIdentifier, url string) error {
	path := l.DataPath(id)

	src, err := os.OpenFile(filepath.Join(path, "remote.json"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open remote description: %w", err)
	}
	defer src.Close()

	var r Remote
	r.URL = url

	e := json.NewEncoder(src)
	if err := e.Encode(r); err != nil {
		return fmt.Errorf("failed to marshall remote description: %w", err)
	}

	return nil
}

func (l *LazyRepository) Remote(id GameIdentifier) (*Remote, error) {
	path := l.DataPath(id)

	src, err := os.OpenFile(filepath.Join(path, "remote.json"), os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open remote description: %w", err)
	}
	defer src.Close()

	var r Remote
	e := json.NewDecoder(src)
	if err := e.Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to marshall remote description: %w", err)
	}

	return &r, nil
}

func (l *LazyRepository) Remove(id GameIdentifier) error {
	path := l.DataPath(id)

	slog.Debug("removing data", "id", id)
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove game folder from the datastore: %w", err)
	}

	return nil
}

func (r *LazyRepository) DataPath(id Identifier) string {
	switch identifier := id.(type) {
	case GameIdentifier:
		return filepath.Join(r.dataRoot, identifier.gameID)
	case BackupIdentifier:
		return filepath.Join(r.dataRoot, identifier.gameID, "hist", identifier.backupID)
	}

	panic("identifier type not supported")
}

func NewEagerRepository(dataRootPath string) (*EagerRepository, error) {
	r, err := NewLazyRepository(dataRootPath)
	if err != nil {
		return nil, err
	}

	return &EagerRepository{
		Repository: r,
		data:       make(map[string]Data),
	}, nil
}

func (r *EagerRepository) Preload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	games, err := r.Repository.All()
	if err != nil {
		return fmt.Errorf("failed to load all data: %w", err)
	}

	for _, g := range games {
		backup, err := r.Repository.AllHist(NewGameIdentifier(g))
		if err != nil {
			return fmt.Errorf("[%s] failed to load hist data: %w", g, err)
		}

		remote, err := r.Repository.Remote(NewGameIdentifier(g))
		if err != nil {
			return fmt.Errorf("[%s] failed to load remote metadata: %w", g, err)
		}

		m, err := r.Repository.Metadata(NewGameIdentifier(g))
		if err != nil {
			return fmt.Errorf("[%s] failed to load metadata: %w", g, err)
		}

		backups := make(map[string]Backup)
		for _, b := range backup {
			info, err := r.Repository.Backup(NewBackupIdentifier(g, b))
			if err != nil {
				return fmt.Errorf("[%s] failed to get backup information: %w", g, err)
			}

			backups[b] = info
		}

		r.data[g] = Data{
			Metadata: m,
			Remote:   remote,
			DataPath: r.DataPath(NewGameIdentifier(g)),
			Backup:   backups,
		}
	}

	return nil
}

func (r *EagerRepository) All() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []string
	for _, g := range r.data {
		res = append(res, g.Metadata.ID)
	}

	return res, nil
}

func (r *EagerRepository) AllHist(id GameIdentifier) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []string
	if d, ok := r.data[id.gameID]; ok {
		for _, b := range d.Backup {
			res = append(res, b.UUID)
		}
	}
	return res, nil
}

func (r *EagerRepository) WriteMetadata(id GameIdentifier, m Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.Repository.WriteMetadata(id, m)
	if err != nil {
		return err
	}

	// reload from disk because of md5
	m, err = r.Repository.Metadata(id)
	if err != nil {
		return err
	}

	d := r.data[id.gameID]
	d.Metadata = m
	r.data[id.gameID] = d

	return nil
}

func (r *EagerRepository) Metadata(id GameIdentifier) (Metadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if d, ok := r.data[id.gameID]; ok {
		return d.Metadata, nil
	}
	return Metadata{}, ErrNotFound
}

func (r *EagerRepository) Backup(id BackupIdentifier) (Backup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if d, ok := r.data[id.gameID]; ok {
		if b, ok := d.Backup[id.backupID]; ok {
			return b, nil
		}
	}
	return Backup{}, ErrNotFound
}

func (r *EagerRepository) SetRemote(id GameIdentifier, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.Repository.SetRemote(id, url)
	if err != nil {
		return err
	}

	d := r.data[id.gameID]
	d.Remote = &Remote{
		URL:    url,
		GameID: d.Metadata.ID,
	}
	r.data[id.gameID] = d

	return nil
}

func (r *EagerRepository) Remove(id GameIdentifier) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.Repository.Remove(id); err != nil {
		return err
	}

	delete(r.data, id.gameID)
	return nil
}
