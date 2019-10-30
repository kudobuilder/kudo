package env

import (
	"os"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
)

// DefaultKudoHome is the default KUDO_HOME. We put .kudo file in the same directory where k8s keeps
// its config files (..kube/config). The place is determined by homedir.HomeDir() method and is different
// from what os.UserHomeDir() returns.
var DefaultKudoHome = filepath.Join(homedir.HomeDir(), ".kudo")
var DefaultKubeConfig = filepath.Join(homedir.HomeDir(), "/.kube/config")

func kudoHome() string {
	if val, ok := os.LookupEnv("KUDO_HOME"); ok {
		return val
	}
	return DefaultKudoHome
}

func kubeConfigHome() string {
	if val, ok := os.LookupEnv("KUBECONFIG"); ok {
		return val
	}
	return DefaultKubeConfig
}

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

// AddFlags binds flags to the given flagset.
func (s *Settings) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar((*string)(&s.Home), "home", kudoHome(), "location of your KUDO config.")
	fs.StringVar(&s.KubeConfig, "kubeconfig", kubeConfigHome(), "Path to your Kubernetes configuration file.")
	fs.StringVarP(&s.Namespace, "namespace", "n", "default", "Target namespace for the object.")
	fs.Int64Var(&s.RequestTimeout, "request-timeout", 0, "Request timeout value, in seconds.  Defaults to 0 (unlimited)")
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
