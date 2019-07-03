package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kudobuilder/kudo/pkg/version"
)

// Client is client used to communicate with KUDO repositories
// it enriches HTTP client with expected headers etc.
type Client struct {
	client *http.Client
}

// Get performs HTTP get on KUDO repository
func (c *Client) Get(href string) (*bytes.Buffer, error) {
	return c.get(href)
}

func (c *Client) get(href string) (*bytes.Buffer, error) {
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
		fmt.Printf("Error when copying response buffer %s", err)
	}
	err = resp.Body.Close()
	if err != nil {
		fmt.Printf("Error when closing the response body %s", err)
	}
	return buf, err
}

// NewClient creates HTTP client
func NewClient() (*Client, error) {
	var client Client
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client, nil
}
