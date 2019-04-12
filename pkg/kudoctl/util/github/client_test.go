package github

import (
	"os"
	"testing"

	"github.com/google/go-github/github"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

// Todo: implementing a mocking interface ala https://github.com/google/go-github/issues/113
// or https://github.com/google/go-github/blob/530b7c552e7576b823c5a3534b6f181ae4340591/github/github_test.go
func newTestGitHubClient() *GithubClient {
	gclient := github.NewClient(nil)
	client := GithubClient{
		client: gclient,
	}
	return &client
}

func TestNewGithubClient(t *testing.T) {

	tests := []struct {
		cred        string
		userEnv     string
		passwordEnv string
		err         string
	}{
		{"", "", "", "no credentials or user environment variable provided"}, // 1
		// {"username:", "", "", "missing github password"}, // Not tested in favor of OTP
		{"username:", "", "", "client test: GET https://api.github.com/user: 401 Requires authentication []"},
		{":password", "", "", "missing github user"},                                                                                                        // 3
		{"anything", "", "", "wrong credentials file format"},                                                                                               // 4
		{"username:password", "", "", "client test: GET https://api.github.com/user: 401 Bad credentials []"},                                               // 5
		{"", "username", "", "no credentials or password environment variable provided"},                                                                    // 6
		{"", "", "password", "no credentials or user environment variable provided"},                                                                        // 7
		{"username:", "", "password", "client test: GET https://api.github.com/user: 401 Bad credentials []"},                                               // 8
		{"anything", "user", "", "wrong credentials format"},                                                                                                // 9
		{"", "user", "password", "client test: GET https://api.github.com/user: 403 Maximum number of login attempts exceeded. Please try again later. []"}, // 10
	}

	for i, tt := range tests {
		i := i
		os.Setenv("GIT_USER", tt.userEnv)
		os.Setenv("GIT_PASSWORD", tt.passwordEnv)
		_, err := NewGithubClient(tt.cred)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}

func TestGithubClient_GetMostRecentFrameworkContentDir(t *testing.T) {

	tests := []struct {
		framework string
		err       string
	}{
		{"", "no framework provided"}, // 1
		{"kafka", ""},                 // 2
		{"non-existing", "getting framework content: incubating framework not found: GET https://api.github.com/repos/kudobuilder/frameworks/contents/repo/incubating/non-existing/versions: 404 Not Found []"}, // 3
	}

	for i, tt := range tests {
		i := i
		gc := newTestGitHubClient()
		_, err := gc.GetMostRecentFrameworkContentDir(tt.framework)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}

func TestGithubClient_GetSpecificFrameworkContentDir(t *testing.T) {

	tests := []struct {
		framework string
		version   string
		err       string
	}{
		{"", "", "no framework provided"},               // 1
		{"kafka", "", "no matching repo version found"}, // 2
		{"kafka", "0", ""},                              // 3
		{"non-existing", "", "getting framework content: incubating framework not found: GET https://api.github.com/repos/kudobuilder/frameworks/contents/repo/incubating/non-existing/versions: 404 Not Found []"}, // 4
	}

	for i, tt := range tests {
		i := i
		gc := newTestGitHubClient()
		vars.PackageVersion = tt.version
		_, err := gc.GetSpecificFrameworkContentDir(tt.framework)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}

func TestGithubClient_GetStableFrameworkContentDir(t *testing.T) {

	tests := []struct {
		framework string
		err       string
	}{
		{"", "no framework provided"}, // 1
		{"non-existing", "stable framework not found: GET https://api.github.com/repos/kudobuilder/frameworks/contents/repo/stable/non-existing/versions: 404 Not Found []"}, // 2
		{"kafka", ""}, // 3
	}

	for i, tt := range tests {
		i := i
		gc := newTestGitHubClient()
		_, err := gc.GetStableFrameworkContentDir(tt.framework)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}

func TestGithubClient_GetIncubatingFrameworkContentDir(t *testing.T) {

	tests := []struct {
		framework string
		err       string
	}{
		{"", "no framework provided"}, // 1
		{"kafka", "incubating framework not found: GET https://api.github.com/repos/kudobuilder/frameworks/contents/repo/incubating/kafka/versions: 404 Not Found []"}, // 2
		{"flink", ""}, // 3
	}

	for i, tt := range tests {
		i := i
		gc := newTestGitHubClient()
		_, err := gc.GetIncubatingFrameworkContentDir(tt.framework)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}

func TestGithubClient_GetFrameworkVersion(t *testing.T) {

	tests := []struct {
		name string
		path string
		err  string
	}{
		{"", "", "no name provided"},                  // 1
		{"kafka", "", "no path provided"},             // 2
		{"", "path/to/framework", "no name provided"}, // 3
		{"kafka", "path/to/framework", "frameworkversion not found: GET https://api.github.com/repos/kudobuilder/frameworks/contents/path/to/framework/kafka-frameworkversion.yaml: 404 Not Found []"}, // 4
		{"kafka", "repo/stable/kafka/versions/0", ""}, // 3
	}

	for i, tt := range tests {
		i := i
		gc := newTestGitHubClient()
		_, err := gc.GetFrameworkVersion(tt.name, tt.path)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.err, err.Error())
			}
		}
	}
}
