package version

import (
	"fmt"
	"os"
	"runtime"
	"strings"
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
