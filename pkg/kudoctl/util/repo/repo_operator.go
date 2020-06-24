package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
)

// Client represents an operator repository
type Client struct {
	Config *Configuration
	Client http.Client
}

func (c *Client) String() string {
	return c.Config.String()
}

// ClientFromSettings retrieves the operator repo for the configured repo in settings
func ClientFromSettings(fs afero.Fs, home kudohome.Home, repoName string) (*Client, error) {
	rc, err := ConfigurationFromSettings(fs, home, repoName)
	if err != nil {
		return nil, err
	}

	return NewClient(rc)
}

// NewClient constructs repository client
func NewClient(conf *Configuration) (*Client, error) {
	_, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %s", conf.URL)
	}

	client := http.NewClient()

	return &Client{
		Config: conf,
		Client: *client,
	}, nil
}

// DownloadIndexFile fetches the index file from a repository.
func (c *Client) DownloadIndexFile() (*IndexFile, error) {
	var indexURL string
	parsedURL, err := url.Parse(c.Config.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing config url: %w", err)
	}
	parsedURL.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(parsedURL.Path, "/"))

	indexURL = parsedURL.String()
	var resp *bytes.Buffer
	if strings.HasPrefix(indexURL, "file:") {
		b, _ := ioutil.ReadFile(parsedURL.Path)
		resp = bytes.NewBuffer(b)
	} else {
		resp, err = c.Client.Get(indexURL)
	}
	if err != nil {
		return nil, fmt.Errorf("getting index url: %w", err)
	}

	indexBytes, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("reading index response: %w", err)
	}

	indexFile, err := ParseIndexFile(indexBytes)
	return indexFile, err
}
