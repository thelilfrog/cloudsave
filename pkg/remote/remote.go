package remote

import (
	"cloudsave/pkg/game"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type (
	Remote struct {
		URL    string `json:"url"`
		GameID string `json:"-"`
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

func All() ([]Remote, error) {
	ds, err := os.ReadDir(datastorepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open the datastore: %w", err)
	}

	var remotes []Remote
	for _, d := range ds {
		content, err := os.ReadFile(filepath.Join(datastorepath, d.Name(), "remote.json"))
		if err != nil {
			continue
		}

		var r Remote
		err = json.Unmarshal(content, &r)
		if err != nil {
			return nil, fmt.Errorf("corrupted datastore: failed to parse %s/remote.json: %w", d.Name(), err)
		}

		content, err = os.ReadFile(filepath.Join(datastorepath, d.Name(), "metadata.json"))
		if err != nil {
			return nil, fmt.Errorf("corrupted datastore: failed to read %s/metadata.json: %w", d.Name(), err)
		}

		var m game.Metadata
		err = json.Unmarshal(content, &m)
		if err != nil {
			return nil, fmt.Errorf("corrupted datastore: failed to parse %s/metadata.json: %w", d.Name(), err)
		}

		r.GameID = m.ID
		remotes = append(remotes, r)
	}
	return remotes, nil
}

func Set(gameID, url string) error {
	r := Remote{
		URL: url,
	}

	f, err := os.OpenFile(filepath.Join(datastorepath, gameID, "remote.json"), os.O_WRONLY|os.O_CREATE, 0740)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	err = e.Encode(r)
	if err != nil {
		return err
	}

	return nil
}
