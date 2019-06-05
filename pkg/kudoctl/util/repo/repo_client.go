package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kudobuilder/kudo/pkg/version"
)

//HTTPClient is client used to communicate with KUDO repositories
//it enriches HTTP client with expected headers etc.
type HTTPClient struct {
	client   *http.Client
}

//Get performs HTTP get on KUDO repository
func (c *HTTPClient) Get(href string) (*bytes.Buffer, error) {
	return c.get(href)
}

func (c *HTTPClient) get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	req, err := http.NewRequest("GET", href, nil)
	if err != nil {
		return buf, err
	}
	req.Header.Set("User-Agent", "KUDO/"+strings.TrimPrefix(version.Get().GitVersion, "v"))

	resp, err := c.client.Do(req)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	err = resp.Body.Close()
	if err != nil {
		fmt.Printf("Error when closing the response body %s", err)
	}
	return buf, err
}

func NewHTTPClient() (*HTTPClient, error) {
	var client HTTPClient
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}

	client.client = &http.Client{Transport: tr}
	return &client, nil
}
