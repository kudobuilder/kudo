package env

import (
	"os"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

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
	// Namespace used when working with Kubernetes
	Namespace string
	// RequestTimeout is the timeout value (in seconds) when making API calls via the KUDO client
	RequestTimeout int64
}

// DefaultSettings initializes the settings to its defaults
var DefaultSettings = &Settings{
	Namespace:      "default",
	RequestTimeout: 0,
}

// envMap maps flag names to envvars
var envMap = map[string]string{
	"home":       "KUDO_HOME",
	"kubeconfig": "KUBECONFIG",
}

// AddFlags binds flags to the given flagset.
func (s *Settings) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar((*string)(&s.Home), "home", DefaultKudoHome, "location of your KUDO config.")
	fs.StringVar(&s.KubeConfig, "kubeconfig", os.Getenv("HOME")+"/.kube/config", "Path to your Kubernetes configuration file.")
	fs.StringVarP(&s.Namespace, "namespace", "n", "default", "Target namespace for the object.")
	fs.Int64Var(&s.RequestTimeout, "request-timeout", 0, "Request timeout value, in seconds.  Defaults to 0 (unlimited)")
}

// Init sets values from the environment.
func (s *Settings) Init(f *pflag.FlagSet) {
	for name, envar := range envMap {
		setFlagFromEnv(name, envar, f)
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
		if err := fs.Set(name, v); err != nil {
			// As all flags are taken from 'envMap', errors aren't expected here.
			panic(err)
		}
	}
}

// OverrideDefault used for deviations from global defaults
func (s *Settings) OverrideDefault(fs *pflag.FlagSet, name, value string) string {
	if fs.Changed(name) {
		return s.Namespace
	}

	return value
}

// GetClient is a helper function that takes the Settings struct and returns a new KUDO Client
func GetClient(s *Settings) (*kudo.Client, error) {
	return kudo.NewClient(s.Namespace, s.KubeConfig, s.RequestTimeout)
}
