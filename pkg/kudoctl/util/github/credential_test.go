package github

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

func TestGetGithubCredentials(t *testing.T) {
	vars.GithubCredentialPath = "/tmp/;"

	testNonExisting := []struct {
		expected string
	}{
		{"credential file: failed to read file: open /tmp/;: no such file or directory"}, // 1
	}

	for _, tt := range testNonExisting {
		_, actual := GetGithubCredentials()
		if actual != nil {
			if actual.Error() != tt.expected {
				t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.expected, actual)
			}
		}
	}

	vars.GithubCredentialPath = ""

	testZero := []struct {
		expected string
	}{
		{"credential file: failed to read file: open : no such file or directory"}, // 1
	}

	for _, tt := range testZero {
		_, actual := GetGithubCredentials()
		if actual.Error() != tt.expected {
			t.Errorf("empty path test:\nexpected: %v\n     got: %v", tt.expected, actual)
		}
	}
}

func TestReadFile(t *testing.T) {

	testNonExisting := []struct {
		err string
	}{
		{"failed to read file: open non-existing: no such file or directory"}, // 1
	}

	for i, tt := range testNonExisting {
		i := i
		_, err := readFile("non-existing")
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestReturnToken(t *testing.T) {

	testNonExisting := []struct {
		token    []byte
		err      string
		expected string
	}{
		{nil, "empty []byte", ""},                                                         // 1
		{[]byte("wrong file content"), "cannot split @", ""},                              // 2
		{[]byte("https://username:password@github.com@m"), "cannot split @", ""},          // 3
		{[]byte("username:password@github.com"), "cannot split https://", ""},             // 4
		{[]byte("https://https://username:password@github.com"), "wrong file format", ""}, // 5
		{[]byte("https://username:password@github.com"), "", "username:password"},         // 5
	}

	for i, tt := range testNonExisting {
		i := i
		actual, err := returnToken(tt.token)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
		if actual != tt.expected {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.expected, actual)
		}
	}
}
