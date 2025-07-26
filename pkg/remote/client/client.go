package client

import (
	"bytes"
	"cloudsave/pkg/game"
	"cloudsave/pkg/remote/obj"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type (
	Client struct {
		baseURL  string
		username string
		password string
	}
)

func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
	}
}

func (c *Client) Hash(gameID string) (string, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "hash")
	if err != nil {
		return "", err
	}

	o, err := c.get(u)
	if err != nil {
		return "", err
	}

	if h, ok := (o.Data).(string); ok {
		return h, nil
	}

	return "", errors.New("invalid payload sent by the server")
}

func (c *Client) Version(gameID string) (int, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "version")
	if err != nil {
		return 0, err
	}

	o, err := c.get(u)
	if err != nil {
		return 0, err
	}

	if h, ok := (o.Data).(float64); ok {
		return int(h), nil
	}

	return 0, errors.New("invalid payload sent by the server")
}

func (c *Client) Push(gameID, archivePath string, m game.Metadata) error {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "games", gameID, "data")
	if err != nil {
		return err
	}

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

	if _, err := io.Copy(f, req.Body); err != nil {
		return fmt.Errorf("an error occured while copying the file from the remote: %w", err)
	}

	if err := os.Rename(archivePath+".part", archivePath); err != nil {
		return fmt.Errorf("failed to move temporary data: %w", err)
	}

	return nil
}

func (c *Client) Ping() bool {
	cli := http.Client{}

	hburl, err := url.JoinPath(c.baseURL, "heartbeat")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	req, err := http.NewRequest("GET", hburl, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	req.SetBasicAuth(c.username, c.password)

	res, err := cli.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to remote:", err)
		return false
	}

	if res.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "cannot connect to remote: server return code", res.StatusCode)
		return false
	}

	return true
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
