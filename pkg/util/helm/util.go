package helm

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

var (
	settings helm_env.EnvSettings
)

func init() {
	settings.Home = helmpath.Home(helm_env.DefaultHelmHome)
	settings.Debug = true
}

// locates the proper url for the tarballed chart
// if version == "", it gets the latest
// order of resolution:
// - current working directory
// - if name is absolute or begins with '.', error out here
// - chart repos in $HELM_HOME
// - URL
func locateChartPath(repoURL, name, version string) (string, error) {
	username := ""
	password := ""
	certFile := ""
	keyFile := ""
	caFile := ""
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if _, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		return abs, nil
	} else {
		fmt.Printf("Error getting %v: %v\n", name, err)
		if os.IsNotExist(err) {
			fmt.Printf("IsNotExist error as expected\n")
		}
	}

	//see if the chart is stored in the local cache
	crepo := filepath.Join(settings.Home.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Getters:  getter.All(settings),
		Username: "",
		Password: "",
	}
	if repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(repoURL, username, password, name, version,
			certFile, keyFile, caFile, getter.All(settings))
		if err != nil {
			return "", err
		}
		name = chartURL
	}

	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	filename, _, err := dl.DownloadTo(name, version, settings.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		return lname, nil
	} else if true { //settings.Debug {
		return filename, err
	}

	return filename, fmt.Errorf("failed to download %q (hint: running `helm repo update` may help)", name)

}

//readFile load a file from the local directory or a remote file with a url.
func readFile(filePath string) ([]byte, error) {
	u, _ := url.Parse(filePath)
	p := getter.All(settings)

	// FIXME: maybe someone handle other protocols like ftp.
	getterConstructor, err := p.ByScheme(u.Scheme)

	if err != nil {
		return ioutil.ReadFile(filePath)
	}

	getter, err := getterConstructor(filePath, "", "", "")
	if err != nil {
		return []byte{}, err
	}
	data, err := getter.Get(filePath)
	return data.Bytes(), err
}
