package github

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"io/ioutil"
	"strings"
)

// GetGithubCredentials stores credentials to GithubCredentials variable.
func GetGithubCredentials() (string, error) {

	dat, err := ioutil.ReadFile(vars.GithubCredentialPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read credential file")
	}

	leading := strings.Split(string(dat), "@")
	trailing := strings.Split(leading[0], "https://")
	credential := trailing[1]

	return credential, nil
}
