package data

import (
	"cloudsave/pkg/game"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Write(gameID, documentRoot string, r io.Reader) error {
	dataFolderPath := filepath.Join(documentRoot, "data", gameID)
	partPath := filepath.Join(dataFolderPath, "data.tar.gz.part")
	finalFilePath := filepath.Join(dataFolderPath, "data.tar.gz")

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

func UpdateMetadata(gameID, documentRoot string, m game.Metadata) error {
	path := filepath.Join(documentRoot, "data", gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	return e.Encode(m)
}
