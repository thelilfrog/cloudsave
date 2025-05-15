package client

import (
	"cloudsave/pkg/remote/obj"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	u, err := url.JoinPath(c.baseURL, "api", "v1", "game", gameID, "hash")
	if err != nil {
		return "", err
	}

	o, err := c.get(u)
	if err != nil {
		return "", err
	}

	if h, ok := (o).(string); ok {
		return h, nil
	}

	return "", errors.New("invalid payload sent by the server")
}

func (c *Client) Version(gameID string) (int, error) {
	u, err := url.JoinPath(c.baseURL, "api", "v1", "game", gameID, "version")
	if err != nil {
		return 0, err
	}

	o, err := c.get(u)
	if err != nil {
		return 0, err
	}

	if h, ok := (o).(int); ok {
		return h, nil
	}

	return 0, errors.New("invalid payload sent by the server")
}

func (c *Client) get(url string) (any, error) {
	cli := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)

	res, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("server returns an unexpected status code: %d %s", res.StatusCode, res.Status)
	}

	var httpObject obj.HTTPObject
	d := json.NewDecoder(res.Body)
	err = d.Decode(&httpObject)
	if err != nil {
		return nil, err
	}

	return httpObject, nil
}

func (c *Client) post() {
	
}