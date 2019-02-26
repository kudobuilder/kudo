package github

import (
	"context"
	"github.com/google/go-github/github" // with go modules disabled
	"os"
	"strings"
)

func NewGithubClient(cred string) (*github.Client, error) {

	// Giving the option to set a Github user and password
	gitUser := os.Getenv("GIT_USER")
	gitPassword := os.Getenv("GIT_PASSWORD")

	result := strings.Split(cred, ":")

	if gitUser == "" {
		gitUser = result[0]
	}

	if gitPassword == "" {
		gitPassword = result[1]
	}

	tp := github.BasicAuthTransport{
		Username: gitUser,
		Password: gitPassword,
	}

	client := github.NewClient(tp.Client())
	ctx := context.Background()
	_, _, err := client.Users.Get(ctx, "")

	// Is this a two-factor auth error? If so, set cred as OTP token
	if _, ok := err.(*github.TwoFactorAuthError); ok {
		tp.OTP = cred
	}

	return client, nil
}
