package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/version"
)

// Client is client used to communicate with KUDO repositories
// it enriches HTTP client with expected headers etc.
type Client struct {
	client *http.Client
}

// Get performs HTTP get on KUDO repository
func (c *Client) Get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("KUDO/%s", strings.TrimPrefix(version.Get().GitVersion, "v")))

	resp, err := c.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		clog.Printf("Error when copying response buffer %s", err)
	}
	err = resp.Body.Close()
	if err != nil {
		clog.Printf("Error when closing the response body %s", err)
	}
	return buf, err
}

// NewClient creates HTTP client
func NewClient() *Client {
	var client Client
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client
}

// IsValidURL returns true if the url is a Parsable URL
func IsValidURL(uri string) bool {
	_, err := url.ParseRequestURI(uri)
	return err == nil
}
