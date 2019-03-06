package github

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"io/ioutil"
	"strings"
)

// GetGithubCredentials stores credentials to GithubCredentials variable.
func GetGithubCredentials() (string, error) {

	credentialFile, err := readFile(vars.GithubCredentialPath)
	if err != nil {
		return "", errors.Wrap(err, "credential file")
	}

	token, err := returnToken(credentialFile)
	if err != nil {
		return "", errors.Wrap(err, "credential file")
	}

	return token, nil
}

// readFile is a wrapper around ioutil.ReadFile and takes a filename and returns the content
func readFile(file string) ([]byte, error) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}
	return dat, nil
}

// returnToken returns the GitHub token of a credential file
func returnToken(dat []byte) (string, error) {
	if dat == nil {
		return "", errors.Errorf("empty []byte")
	}
	leading := strings.Split(string(dat), "@")
	if len(leading) <= 1 || len(leading) > 2 {
		return "", errors.Errorf("cannot split @")
	}
	trailing := strings.Split(leading[0], "https://")
	if len(trailing) < 2 {
		return "", errors.Errorf("cannot split https://")
	}
	token := trailing[1]
	if token == "" {
		return "", errors.Errorf("wrong file format")
	}
	return token, nil
}
