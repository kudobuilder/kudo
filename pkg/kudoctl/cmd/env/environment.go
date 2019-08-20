package env

import (
	"os"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/env/kudohome"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
)

// DefaultKudoHome is the default KUDO_HOME.
var DefaultKudoHome = filepath.Join(homedir.HomeDir(), ".kudo")

// Settings defines global variables and settings
type Settings struct {
	// KubeConfig is the path to an explicit kubeconfig file. This overwrites the value in $KUBECONFIG
	KubeConfig string
	// Home is the local path to kudo home directory
	Home kudohome.Home
	// Debug indicates whether or not Kudo is running in Debug mode.
	Debug bool
	// Repo is the name of the repo to use if not default
	Repo string
}

// envMap maps flag names to envvars
var envMap = map[string]string{
	"debug":      "KUDO_DEBUG",
	"home":       "KUDO_HOME",
	"kubeconfig": "KUBECONFIG",
	"repo":       "KUDO_REPO",
}

// AddFlags binds flags to the given flagset.
func (s *Settings) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar((*string)(&s.Home), "home", DefaultKudoHome, "location of your KUDO config.")
	fs.BoolVar(&s.Debug, "debug", false, "enable verbose output")
	fs.StringVar(&s.KubeConfig, "kubeconfig", os.Getenv("HOME")+"/.kube/config", "Path to your Kubernetes configuration file")
	fs.StringVar(&s.Repo, "repo", "testing", "Name of repo to use")
}

// Init sets values from the environment.
func (s *Settings) Init(fs *pflag.FlagSet) {
	for name, envar := range envMap {
		setFlagFromEnv(name, envar, fs)
	}
}

// setFlagFromEnv looks up and sets a flag if the corresponding environment variable changed.
// if the flag with the corresponding name was set during fs.Parse(), then the environment
// variable is ignored.
func setFlagFromEnv(name, envar string, fs *pflag.FlagSet) {
	if fs.Changed(name) {
		return
	}
	if v, ok := os.LookupEnv(envar); ok {
		fs.Set(name, v)
	}
}
