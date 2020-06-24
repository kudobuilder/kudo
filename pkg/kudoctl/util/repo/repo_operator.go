package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
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
	parsedURL, err := url.Parse(c.Config.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing config url: %w", err)
	}
	h := make(map[string]bool)
	return c.downloadIndexFile(parsedURL, h)
}

func (c *Client) downloadIndexFile(url *url.URL, urlHistory map[string]bool) (*IndexFile, error) {
	var resp *bytes.Buffer
	var err error
	// we need the index.yaml at the url provided
	url.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(url.Path, "/"))
	if val, ok := urlHistory[url.String()]; ok {
		// if we have seen the url previous we don't process it
		clog.V(1).Printf("duplicate url %v ignored", val)
		return nil, nil
	}
	urlHistory[url.String()] = true

	if url.Scheme == "file" || strings.HasPrefix(url.String(), "file:") {
		b, err := ioutil.ReadFile(url.Path)
		if err != nil {
			return nil, err
		}
		resp = bytes.NewBuffer(b)
	} else {
		resp, err = c.Client.Get(url.String())
	}
	if err != nil {
		return nil, fmt.Errorf("getting index url: %w", err)
	}

	indexBytes, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("reading index response: %w", err)
	}

	indexFile, err := ParseIndexFile(indexBytes)
	for _, include := range indexFile.Includes {
		iURL, err := url.Parse(include)
		if err != nil {
			return nil, clog.Errorf("unable to parse include url for %s", include)
		}
		nextIndex, err := c.downloadIndexFile(iURL, urlHistory)
		if err != nil {
			return nil, err
		}
		err = c.Merge(indexFile, nextIndex)
		if err != nil {
			return nil, err
		}
	}

	return indexFile, err
}

// Merge combines the Entries of 2 index files.   The first index file is the master
// the second is merged into the first.  Any duplicates are ignored.
func (c *Client) Merge(index *IndexFile, mergeIndex *IndexFile) error {
	// index is the master, any dups in the merged in index will have what is local replace those entries
	for _, pvs := range mergeIndex.Entries {
		for _, pv := range pvs {
			err := index.AddPackageVersion(pv)
			if errors.Is(err, &DuplicateError{}) {
				clog.V(1).Printf("ignoring duplicate for %q: appver: %v, opver: %v", pv.Name, pv.AppVersion, pv.OperatorVersion)
				continue
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}
