package repo

import (
	"bytes"
	"fmt"
	"github.com/kudobuilder/kudo/pkg/version"
	"io"
	"net/http"
	"strings"
)

//RepoClient is the default HTTP(/S) backend handler
type RepoClient struct {
	client   *http.Client
	username string
	password string
}

//SetCredentials sets the credentials for the RepoClient
func (g *RepoClient) SetCredentials(username, password string) {
	g.username = username
	g.password = password
}

//Get performs a Get from repo.Getter and returns the body.
func (g *RepoClient) Get(href string) (*bytes.Buffer, error) {
	return g.get(href)
}

func (g *RepoClient) get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	// Set a KUDO specific user agent so that a repo server and metrics can
	// separate helm calls from other tools interacting with repos.
	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", "KUDO/"+strings.TrimPrefix(version.Get().GitVersion, "v"))

	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("Failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	resp.Body.Close()
	return buf, err
}

// NewHTTPGetter constructs a valid http/https client as HttpGetter
func NewHTTPRepoClient(URL string) (*RepoClient, error) {
	var client RepoClient
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client, nil
}
