package remote

import (
	"encoding/json"
	"errors"
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

var (
	ErrNoRemote error = errors.New("no remote found for this game")
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

func One(gameID string) (Remote, error) {
	content, err := os.ReadFile(filepath.Join(datastorepath, gameID, "remote.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Remote{}, ErrNoRemote
		}
		return Remote{}, err
	}

	var r Remote
	err = json.Unmarshal(content, &r)
	if err != nil {
		return Remote{}, fmt.Errorf("corrupted datastore: failed to parse %s/remote.json: %w", gameID, err)
	}

	r.GameID = gameID
	return r, nil
}

func Set(gameID, url string) error {
	r := Remote{
		URL: url,
	}

	f, err := os.OpenFile(filepath.Join(datastorepath, gameID, "remote.json"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0740)
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
