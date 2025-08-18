package client

import (
	"bytes"
	"cloudsave/pkg/remote/obj"
	"cloudsave/pkg/repository"
	customtime "cloudsave/pkg/tools/time"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
)

type (
	Client struct {
		baseURL  string
		username string
		password string
	}

	Information struct {
		Version        string `json:"version"`
		APIVersion     int    `json:"api_version"`
		GoVersion      string `json:"go_version"`
		OSName         string `json:"os_name"`
		OSArchitecture string `json:"os_architecture"`
	}
)

var (
	ErrNotFound     error = errors.New("not found")
	ErrUnauthorized error = errors.New("unauthorized (HTTP Error 401)")
)

func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
	}
}

func (c *Client) Exists(gameID string) (bool, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "metadata")
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("HEAD", u, nil)
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(c.username, c.password)

	cli := http.Client{}

	r, err := cli.Do(req)
	if err != nil {
		return false, err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	}

	return false, fmt.Errorf("an error occured: server response: %s", r.Status)
}

func (c *Client) Version() (Information, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "version")
	if err != nil {
		return Information{}, err
	}

	o, err := c.get(u)
	if err != nil {
		return Information{}, err
	}

	if info, ok := (o.Data).(map[string]any); ok {
		i := Information{
			Version:        info["version"].(string),
			APIVersion:     int(info["api_version"].(float64)),
			GoVersion:      info["go_version"].(string),
			OSName:         info["os_name"].(string),
			OSArchitecture: info["os_architecture"].(string),
		}
		return i, nil
	}

	return Information{}, errors.New("invalid payload sent by the server")
}

// Deprecated: use c.Metadata instead
func (c *Client) Hash(gameID string) (string, error) {
	m, err := c.Metadata(gameID)
	if err != nil {
		return "", err
	}
	return m.MD5, nil
}

func (c *Client) Metadata(gameID string) (repository.Metadata, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "metadata")
	if err != nil {
		return repository.Metadata{}, err
	}

	o, err := c.get(u)
	if err != nil {
		return repository.Metadata{}, err
	}

	if m, ok := (o.Data).(map[string]any); ok {
		gm := repository.Metadata{
			ID:      m["id"].(string),
			Name:    m["name"].(string),
			Version: int(m["version"].(float64)),
			Date:    customtime.MustParse(time.RFC3339, m["date"].(string)),
			MD5:     m["md5"].(string),
		}
		return gm, nil
	}

	return repository.Metadata{}, errors.New("invalid payload sent by the server")
}

func (c *Client) PushSave(archivePath string, m repository.Metadata) error {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", m.ID, "data")
	if err != nil {
		return err
	}

	return c.push(u, archivePath, m)
}

func (c *Client) PushBackup(archiveMetadata repository.Backup, m repository.Metadata) error {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", m.ID, "hist", archiveMetadata.UUID, "data")
	if err != nil {
		return err
	}

	return c.push(u, archiveMetadata.ArchivePath, m)
}

func (c *Client) ListArchives(gameID string) ([]string, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "hist")
	if err != nil {
		return nil, err
	}

	o, err := c.get(u)
	if err != nil {
		return nil, err
	}

	if m, ok := (o.Data).([]any); ok {
		var res []string
		for _, uuid := range m {
			res = append(res, uuid.(string))
		}
		return res, nil
	}

	return nil, errors.New("invalid payload sent by the server")
}

func (c *Client) ArchiveInfo(gameID, uuid string) (repository.Backup, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "hist", uuid, "info")
	if err != nil {
		return repository.Backup{}, err
	}

	o, err := c.get(u)
	if err != nil {
		return repository.Backup{}, err
	}

	if m, ok := (o.Data).(map[string]any); ok {
		b := repository.Backup{
			UUID:      m["uuid"].(string),
			CreatedAt: customtime.MustParse(time.RFC3339, m["created_at"].(string)),
			MD5:       m["md5"].(string),
		}
		return b, nil
	}

	return repository.Backup{}, errors.New("invalid payload sent by the server")
}

func (c *Client) Pull(gameID, archivePath string) error {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "data")
	if err != nil {
		return err
	}

	cli := http.Client{}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)

	f, err := os.OpenFile(archivePath+".part", os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	res, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to remote: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot connect to remote: server return code: %s", res.Status)
	}

	bar := progressbar.DefaultBytes(
		res.ContentLength,
		"Pulling...",
	)
	defer bar.Close()

	if _, err := io.Copy(io.MultiWriter(f, bar), res.Body); err != nil {
		return fmt.Errorf("an error occured while copying the file from the remote: %w", err)
	}

	if err := os.Rename(archivePath+".part", archivePath); err != nil {
		return fmt.Errorf("failed to move temporary data: %w", err)
	}

	return nil
}

func (c *Client) PullBackup(gameID, uuid, archivePath string) error {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "hist", uuid, "data")
	if err != nil {
		return err
	}

	cli := http.Client{}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)

	f, err := os.OpenFile(archivePath+".part", os.O_CREATE|os.O_WRONLY, 0740)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	res, err := cli.Do(req)
	if err != nil {
		f.Close()
		return fmt.Errorf("cannot connect to remote: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		f.Close()
		return fmt.Errorf("cannot connect to remote: server return code: %s", res.Status)
	}

	bar := progressbar.DefaultBytes(
		res.ContentLength,
		"Pulling...",
	)
	defer bar.Close()

	if _, err := io.Copy(io.MultiWriter(f, bar), res.Body); err != nil {
		f.Close()
		return fmt.Errorf("an error occured while copying the file from the remote: %w", err)
	}
	f.Close()

	if err := os.Rename(archivePath+".part", archivePath); err != nil {
		return fmt.Errorf("failed to move temporary data: %w", err)
	}

	return nil
}

func (c *Client) Ping() error {
	cli := http.Client{}

	hburl, err := url.JoinPath(c.baseURL, "heartbeat")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", hburl, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)

	res, err := cli.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot connect to remote: server return code %s", res.Status)
	}

	return nil
}

func (c *Client) All() ([]repository.Metadata, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games")
	if err != nil {
		return nil, err
	}

	o, err := c.get(u)
	if err != nil {
		return nil, err
	}

	if games, ok := (o.Data).([]any); ok {
		var res []repository.Metadata
		for _, g := range games {
			if v, ok := g.(map[string]any); ok {
				gm := repository.Metadata{
					ID:      v["id"].(string),
					Name:    v["name"].(string),
					Version: int(v["version"].(float64)),
					Date:    customtime.MustParse(time.RFC3339, v["date"].(string)),
					MD5:     v["md5"].(string),
				}
				res = append(res, gm)
			}
		}

		return res, nil
	}

	return nil, errors.New("invalid payload sent by the server")
}

func (c *Client) get(url string) (obj.HTTPObject, error) {
	cli := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return obj.HTTPObject{}, err
	}

	req.SetBasicAuth(c.username, c.password)

	res, err := cli.Do(req)
	if err != nil {
		return obj.HTTPObject{}, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return obj.HTTPObject{}, ErrNotFound
	}

	if res.StatusCode == 401 {
		return obj.HTTPObject{}, ErrUnauthorized
	}

	if res.StatusCode != 200 {
		return obj.HTTPObject{}, fmt.Errorf("server returns an unexpected status code: %d %s (expected 200)", res.StatusCode, res.Status)
	}

	var httpObject obj.HTTPObject
	d := json.NewDecoder(res.Body)
	err = d.Decode(&httpObject)
	if err != nil {
		return obj.HTTPObject{}, err
	}

	return httpObject, nil
}

func (c *Client) push(u, archivePath string, m repository.Metadata) error {
	f, err := os.OpenFile(archivePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("payload", "data.tar.gz")
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	writer.WriteField("name", m.Name)
	writer.WriteField("version", strconv.Itoa(m.Version))
	writer.WriteField("date", m.Date.Format(time.RFC3339))

	if err := writer.Close(); err != nil {
		return err
	}

	cli := http.Client{}

	req, err := http.NewRequest("POST", u, buf)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		return fmt.Errorf("server returns an unexpected status code: %s (expected 201)", res.Status)
	}

	return nil
}
