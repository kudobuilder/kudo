package repo

import (
	"bytes"
	"fmt"
	"github.com/kudobuilder/kudo/pkg/version"
	"io"
	"net/http"
	"strings"
)

//Client is the default HTTP(/S) backend handler
type Client struct {
	client   *http.Client
	username string
	password string
}

//SetCredentials sets the credentials for the RepoClient
func (c *Client) SetCredentials(username, password string) {
	c.username = username
	c.password = password
}

//Get performs a Get from repo.Getter and returns the body.
func (c *Client) Get(href string) (*bytes.Buffer, error) {
	return c.get(href)
}

func (c *Client) get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	// Set a KUDO specific user agent so that a repo server and metrics can
	// separate helm calls from other tools interacting with repos.
	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", "KUDO/"+strings.TrimPrefix(version.Get().GitVersion, "v"))

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	resp.Body.Close()
	return buf, err
}

// NewHTTPClient constructs a valid http/https client as HttpClient
func NewHTTPClient(URL string) (*Client, error) {
	var client Client
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client, nil
}
