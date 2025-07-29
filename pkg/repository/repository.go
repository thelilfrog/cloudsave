package repository

import (
	"cloudsave/pkg/tools/id"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type (
	Metadata struct {
		ID      string    `json:"id"`
		Name    string    `json:"name"`
		Path    string    `json:"path"`
		Version int       `json:"version"`
		Date    time.Time `json:"date"`
	}
)

var (
	roaming       string
	datastorepath string
)

func init() {
	var err error
	roaming, err = os.UserConfigDir()
	if err != nil {
		panic("failed to get user config path: " + err.Error())
	}

	datastorepath = filepath.Join(roaming, "cloudsave", "data")
	err = os.MkdirAll(datastorepath, 0740)
	if err != nil {
		panic("cannot make the datastore:" + err.Error())
	}
}

func Add(name, path string) (Metadata, error) {
	m := Metadata{
		ID:   id.New(),
		Name: name,
		Path: path,
	}

	err := os.MkdirAll(filepath.Join(datastorepath, m.ID), 0740)
	if err != nil {
		panic("cannot make directory for the game:" + err.Error())
	}

	f, err := os.OpenFile(filepath.Join(datastorepath, m.ID, "metadata.json"), os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return Metadata{}, fmt.Errorf("cannot open the metadata file in the datastore: %w", err)
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(m)
	if err != nil {
		return Metadata{}, fmt.Errorf("cannot write into the metadata file in the datastore: %w", err)
	}

	return m, nil
}

func Register(m Metadata, path string) error {
	m.Path = path

	err := os.MkdirAll(filepath.Join(datastorepath, m.ID), 0740)
	if err != nil {
		panic("cannot make directory for the game:" + err.Error())
	}

	f, err := os.OpenFile(filepath.Join(datastorepath, m.ID, "metadata.json"), os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("cannot open the metadata file in the datastore: %w", err)
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(m)
	if err != nil {
		return fmt.Errorf("cannot write into the metadata file in the datastore: %w", err)
	}

	return nil
}

func All() ([]Metadata, error) {
	ds, err := os.ReadDir(datastorepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open the datastore: %w", err)
	}

	var datastore []Metadata
	for _, d := range ds {
		content, err := os.ReadFile(filepath.Join(datastorepath, d.Name(), "metadata.json"))
		if err != nil {
			continue
		}

		var m Metadata
		err = json.Unmarshal(content, &m)
		if err != nil {
			return nil, fmt.Errorf("corrupted datastore: failed to parse %s/metadata.json: %w", d.Name(), err)
		}

		datastore = append(datastore, m)
	}
	return datastore, nil
}

func One(gameID string) (Metadata, error) {
	_, err := os.ReadDir(datastorepath)
	if err != nil {
		return Metadata{}, fmt.Errorf("cannot open the datastore: %w", err)
	}

	content, err := os.ReadFile(filepath.Join(datastorepath, gameID, "metadata.json"))
	if err != nil {
		return Metadata{}, fmt.Errorf("game not found: %w", err)
	}

	var m Metadata
	err = json.Unmarshal(content, &m)
	if err != nil {
		return Metadata{}, fmt.Errorf("corrupted datastore: failed to parse %s/metadata.json: %w", gameID, err)
	}

	return m, nil
}

func Archive(gameID string) error {
	path := filepath.Join(datastorepath, gameID, "data.tar.gz")

	// open old
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open old file: %w", err)
	}
	defer f.Close()

	histDirPath := filepath.Join(datastorepath, gameID, "hist")
	if err := os.MkdirAll(histDirPath, 0740); err != nil {
		return fmt.Errorf("failed to make 'hist' directory")
	}

	d, err := os.ReadDir(histDirPath)
	if err != nil {
		return fmt.Errorf("failed to open 'hist' directory")
	}

	// keep the dir under 6 files
	if len(d) > 5 {
		var oldest *fs.FileInfo
		for _, hfile := range d {
			finfo, err := hfile.Info()
			if err != nil {
				return fmt.Errorf("failed to read backup file: %w", err)
			}

			if oldest == nil {
				oldest = &finfo
				continue
			}

			if finfo.ModTime().Before((*oldest).ModTime()) {
				oldest = &finfo
			}
		}

		if err := os.Remove((*oldest).Name()); err != nil {
			return fmt.Errorf("failed to remove the oldest backup file: %w", err)
		}
	}

	// open new
	nf, err := os.OpenFile(filepath.Join(datastorepath, gameID, "hist", time.Now().Format("2006-01-02T15-04-05Z07-00")+".data.tar.gz"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open new file: %w", err)
	}
	defer nf.Close()

	// copy
	if _, err := io.Copy(nf, f); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

func DatastorePath() string {
	return datastorepath
}

func Remove(gameID string) error {
	err := os.RemoveAll(filepath.Join(datastorepath, gameID))
	if err != nil {
		return err
	}
	return nil
}

func Hash(gameID string) (string, error) {
	path := filepath.Join(datastorepath, gameID, "data.tar.gz")

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum), nil
}

func Version(gameID string) (int, error) {
	path := filepath.Join(datastorepath, gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var metadata Metadata
	d := json.NewDecoder(f)
	err = d.Decode(&metadata)
	if err != nil {
		return 0, err
	}

	return metadata.Version, nil
}

func SetVersion(gameID string, version int) error {
	path := filepath.Join(datastorepath, gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	var metadata Metadata
	d := json.NewDecoder(f)
	err = d.Decode(&metadata)
	if err != nil {
		f.Close()
		return err
	}

	f.Close()

	metadata.Version = version

	f, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0740)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(metadata)
	if err != nil {
		return err
	}

	return nil
}

func SetDate(gameID string, dt time.Time) error {
	path := filepath.Join(datastorepath, gameID, "metadata.json")

	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	var metadata Metadata
	d := json.NewDecoder(f)
	err = d.Decode(&metadata)
	if err != nil {
		f.Close()
		return err
	}

	f.Close()

	metadata.Date = dt

	f, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0740)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(metadata)
	if err != nil {
		return err
	}

	return nil
}
