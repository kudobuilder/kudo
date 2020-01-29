package env

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
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

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envars {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("failed to set env var %s=%s: %v", k, v, err)
				}
			}

			flags := pflag.NewFlagSet("testing", pflag.ContinueOnError)

			settings := &Settings{}
			settings.AddFlags(flags)

			if err := flags.Parse(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if settings.Home != kudohome.Home(tt.home) {
				t.Errorf("expected home %q, got %q", tt.home, settings.Home)
			}
			if settings.KubeConfig != tt.kconfig {
				t.Errorf("expected kubeconfig %q, got %q", tt.kconfig, settings.KubeConfig)
			}
			if settings.RequestTimeout != tt.requesttimeout {
				t.Errorf("expected request-timeout %d, got %d", tt.requesttimeout, settings.RequestTimeout)
			}
			resetEnv(t, tt.envars)
		})
	}
}

func resetEnv(t *testing.T, envars map[string]string) func(t *testing.T) {
	origEnv := os.Environ()

	// clear local envvars of test envs
	for e := range envars {
		if err := os.Unsetenv(e); err != nil {
			t.Fatalf("failed to unset env var %s: %v", e, err)
		}
	}

	return func(t *testing.T) {
		for _, pair := range origEnv {
			kv := strings.SplitN(pair, "=", 2)
			if err := os.Setenv(kv[0], kv[1]); err != nil {
				t.Fatalf("failed to reset env var %s=%s: %v", kv[0], kv[1], err)
			}
		}
	}
}
