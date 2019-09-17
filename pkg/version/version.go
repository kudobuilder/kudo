package version

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver"
)

// Info contains versioning information.
type Info struct {
	GitVersion string `json:"gitVersion"`
	GitCommit  string `json:"gitCommit"`
	BuildDate  string `json:"buildDate"`
	GoVersion  string `json:"goVersion"`
	Compiler   string `json:"compiler"`
	Platform   string `json:"platform"`
}

// String returns info as a human-friendly version string.
func (info Info) String() string {
	return info.GitVersion
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Info {
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in pkg/version/base.go
	// developer fallback for version

	// this only happens when running from a build.  Release runs ARE correct.
	if strings.Contains(gitVersion, "$Format") {
		// on dev box, lets use a env var for version
		gitVersion = os.Getenv("KUDO_DEV_VERSION")
		if gitVersion == "" {
			gitVersion = "dev"
		}
		gitCommit = "dev"
		//TODO (kensipe): add debug message!
	}

	return Info{
		GitVersion: gitVersion,
		GitCommit:  gitCommit,
		BuildDate:  buildDate,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// Error is an error for versions in case it is desired to check an error for this type
type Error struct {
	Component       string
	ExpectedVersion string
	Version         string
}

// Error is a required function for type Error
func (e Error) Error() string {
	return fmt.Sprintf("expected version: %s, found vresion: %s", e.ExpectedVersion, e.Version)
}

//Valid returns nil if expected version is meet by actual using solely Major.Minor
func Valid(component string, actual *semver.Version, expected *semver.Version) error {
	if actual.Major() < expected.Major() || actual.Minor() < expected.Minor() {
		return Error{
			Component:       component,
			ExpectedVersion: expected.String(),
			Version:         actual.String(),
		}
	}

	return nil
}

// Clean returns version without a prefixed v if it exists
func Clean(ver string) string {
	if strings.HasPrefix(ver, "v") {
		return ver[1:]
	}
	return ver
}
