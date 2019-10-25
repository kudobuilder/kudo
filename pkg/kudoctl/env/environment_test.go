package env

import (
	"os"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"

	"github.com/spf13/pflag"
)

func TestEnvSettings(t *testing.T) {
	tests := []struct {
		name string

		// input
		args   []string
		envars map[string]string

		// expected values
		home, kconfig  string
		requesttimeout int64
	}{
		{
			name:    "defaults",
			args:    []string{},
			home:    DefaultKudoHome,
			kconfig: os.Getenv("HOME") + "/.kube/config",
		},
		{
			name:    "with flags set",
			args:    []string{"--home", "/foo", "--kubeconfig", "/bar"},
			home:    "/foo",
			kconfig: "/bar",
		},
		{
			name:    "with ENV set",
			args:    []string{},
			envars:  map[string]string{"KUDO_HOME": "/bar", "KUBECONFIG": "/foo"},
			home:    "/bar",
			kconfig: "/foo",
		},
		{
			name:    "with flags and ENV set",
			args:    []string{"--home", "/foo", "--kubeconfig", "/bar"},
			envars:  map[string]string{"KUDO_HOME": "/bar", "KUBECONFIG": "/foo"},
			home:    "/foo",
			kconfig: "/bar",
		},
		{
			name:           "with request timeout set",
			args:           []string{"--request-timeout", "5"},
			home:           DefaultKudoHome,
			kconfig:        os.Getenv("HOME") + "/.kube/config",
			requesttimeout: 5,
		},
	}

	allEnvvars := map[string]string{
		"KUDO_HOME":  "",
		"KUBECONFIG": "",
	}

	resetOrigEnv := resetEnv(allEnvvars)
	defer resetOrigEnv()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envars {
				os.Setenv(k, v)
			}

			flags := pflag.NewFlagSet("testing", pflag.ContinueOnError)

			settings := &Settings{}
			settings.AddFlags(flags)
			flags.Parse(tt.args)

			settings.Init(flags)

			if settings.Home != kudohome.Home(tt.home) {
				t.Errorf("expected home %q, got %q", tt.home, settings.Home)
			}
			if settings.KubeConfig != tt.kconfig {
				t.Errorf("expected kubeconfig %q, got %q", tt.kconfig, settings.KubeConfig)
			}
			if settings.RequestTimeout != tt.requesttimeout {
				t.Errorf("expected request-timeout %d, got %d", tt.requesttimeout, settings.RequestTimeout)
			}
			resetEnv(tt.envars)
		})
	}
}

func resetEnv(envars map[string]string) func() {
	origEnv := os.Environ()

	// clear local envvars of test envs
	for e := range envars {
		os.Unsetenv(e)
	}

	return func() {
		for _, pair := range origEnv {
			kv := strings.SplitN(pair, "=", 2)
			os.Setenv(kv[0], kv[1])
		}
	}
}
