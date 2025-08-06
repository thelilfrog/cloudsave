package data

import (
	"cloudsave/pkg/repository"
	"cloudsave/pkg/tools/hash"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type (
	cache map[string]cachedInfo

	cachedInfo struct {
		MD5     string
		Version int
	}
)

var (
	ErrBackupNotExists error = errors.New("backup not found")

	// singleton
	hashCacheMu sync.RWMutex
	hashCache   cache = make(map[string]cachedInfo)
)

func (c cache) Get(gameID string) (cachedInfo, bool) {
	hashCacheMu.RLock()
	defer hashCacheMu.RUnlock()

	if v, ok := c[gameID]; ok {
		return v, true
	}
	return cachedInfo{}, false
}

func (c cache) Register(gameID string, v cachedInfo) {
	hashCacheMu.Lock()
	defer hashCacheMu.Unlock()

	c[gameID] = v
}

func (c cache) Remove(gameID string) {
	hashCacheMu.Lock()
	defer hashCacheMu.Unlock()

	delete(c, gameID)
}

func Write(gameID, documentRoot string, r io.Reader) error {
	dataFolderPath := filepath.Join(documentRoot, "data", gameID)
	partPath := filepath.Join(dataFolderPath, "data.tar.gz.part")
	finalFilePath := filepath.Join(dataFolderPath, "data.tar.gz")

	if err := makeDataFolder(gameID, documentRoot); err != nil {
		return err
	}

	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		if err := os.Remove(partPath); err != nil {
			return fmt.Errorf("failed to write the file and cannot clean the folder: %w", err)
		}
		return fmt.Errorf("failed to write the file: %w", err)
	}
	f.Close()

	if err := os.Rename(partPath, finalFilePath); err != nil {
		return err
	}

	hashCache.Remove(gameID)
	return nil
}

func WriteHist(gameID, documentRoot, uuid string, r io.Reader) error {
	dataFolderPath := filepath.Join(documentRoot, "data", gameID, "hist", uuid)
	partPath := filepath.Join(dataFolderPath, "data.tar.gz.part")
	finalFilePath := filepath.Join(dataFolderPath, "data.tar.gz")

	if err := makeDataFolder(gameID, documentRoot); err != nil {
		return err
	}

	if err := os.MkdirAll(dataFolderPath, 0740); err != nil {
		return err
	}

	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		if err := os.Remove(partPath); err != nil {
			return fmt.Errorf("failed to write the file and cannot clean the folder: %w", err)
		}
		return fmt.Errorf("failed to write the file: %w", err)
	}
	f.Close()

	if err := os.Rename(partPath, finalFilePath); err != nil {
		return err
	}

	return nil
}

func UpdateMetadata(gameID, documentRoot string, m repository.Metadata) error {
	if err := makeDataFolder(gameID, documentRoot); err != nil {
		return err
	}
	path := filepath.Join(documentRoot, "data", gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	return e.Encode(m)
}

func ArchiveInfo(gameID, documentRoot, uuid string) (repository.Backup, error) {
	dataFolderPath := filepath.Join(documentRoot, "data", gameID, "hist", uuid, "data.tar.gz")
	cacheID := gameID + ":" + uuid

	finfo, err := os.Stat(dataFolderPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return repository.Backup{}, ErrBackupNotExists
		}
		return repository.Backup{}, err
	}

	v, err := getVersion(gameID, documentRoot)
	if err != nil {
		return repository.Backup{}, fmt.Errorf("failed to read game metadata: %w", err)
	}

	if m, ok := hashCache.Get(cacheID); ok {
		return repository.Backup{
			CreatedAt: finfo.ModTime(),
			UUID:      uuid,
			MD5:       m.MD5,
		}, nil
	}

	h, err := hash.FileMD5(dataFolderPath)
	if err != nil {
		return repository.Backup{}, fmt.Errorf("failed to calculate file md5: %w", err)
	}

	hashCache.Register(cacheID, cachedInfo{
		Version: v,
		MD5:     h,
	})

	return repository.Backup{
		CreatedAt: finfo.ModTime(),
		UUID:      uuid,
		MD5:       h,
	}, nil
}

func Hash(gameID, documentRoot string) (string, error) {
	path := filepath.Clean(filepath.Join(documentRoot, "data", gameID))

	sdir, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !sdir.IsDir() {
		return "", err
	}

	v, err := getVersion(gameID, documentRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read game metadata: %w", err)
	}

	if m, ok := hashCache.Get(gameID); ok {
		if v == m.Version {
			return m.MD5, nil
		}
	}

	path = filepath.Join(path, "data.tar.gz")

	h, err := hash.FileMD5(path)
	if err != nil {
		return "", err
	}

	hashCache.Register(gameID, cachedInfo{
		Version: v,
		MD5:     h,
	})

	return h, nil
}

func getVersion(gameID, documentRoot string) (int, error) {
	path := filepath.Join(documentRoot, "data", gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	var m repository.Metadata
	if err := d.Decode(&m); err != nil {
		return 0, err
	}

	return m.Version, nil
}

func makeDataFolder(gameID, documentRoot string) error {
	if err := os.MkdirAll(filepath.Join(documentRoot, "data", gameID), 0740); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(documentRoot, "data", gameID, "hist"), 0740); err != nil {
		return err
	}

	return nil
}
