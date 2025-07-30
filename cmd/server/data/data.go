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
)

var (
	ErrBackupNotExists error = errors.New("backup not found")
)

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

	finfo, err := os.Stat(dataFolderPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return repository.Backup{}, ErrBackupNotExists
		}
		return repository.Backup{}, err
	}

	h, err := hash.FileMD5(dataFolderPath)
	if err != nil {
		return repository.Backup{}, fmt.Errorf("failed to calculate file md5: %w", err)
	}

	return repository.Backup{
		CreatedAt: finfo.ModTime(),
		UUID:      uuid,
		MD5:       h,
	}, nil
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
