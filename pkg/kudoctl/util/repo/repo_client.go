package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kudobuilder/kudo/pkg/version"
)

//HTTPClient is the default HTTP(/S) backend handler
type HTTPClient struct {
	client   *http.Client
	username string
	password string
}

//SetCredentials sets the credentials for the RepoClient
func (c *HTTPClient) SetCredentials(username, password string) {
	c.username = username
	c.password = password
}

//Get performs a Get from repo.Getter and returns the body.
func (c *HTTPClient) Get(href string) (*bytes.Buffer, error) {
	return c.get(href)
}

func (c *HTTPClient) get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	// Set a KUDO specific user agent so that a repo server and metrics can
	// separate KUDO calls from other tools interacting with repos.
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
func NewHTTPClient(URL string) (*HTTPClient, error) {
	var client HTTPClient
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client, nil
}
