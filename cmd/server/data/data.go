package data

import (
	"cloudsave/pkg/repository"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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

func WriteHist(gameID, documentRoot string, dt time.Time, r io.Reader) error {
	dataFolderPath := filepath.Join(documentRoot, "data", gameID, "hist")
	partPath := filepath.Join(dataFolderPath, dt.Format("2006-01-02T15-04-05Z07-00")+".data.tar.gz.part")
	finalFilePath := filepath.Join(dataFolderPath, dt.Format("2006-01-02T15-04-05Z07-00")+".data.tar.gz")

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

func makeDataFolder(gameID, documentRoot string) error {
	if err := os.MkdirAll(filepath.Join(documentRoot, "data", gameID), 0740); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(documentRoot, "data", gameID, "hist"), 0740); err != nil {
		return err
	}

	return nil
}
