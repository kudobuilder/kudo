package env

import (
	"os"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/env/kudohome"

	"github.com/spf13/pflag"
)

func TestEnvSettings(t *testing.T) {
	tests := []struct {
		name string

		// input
		args   []string
		envars map[string]string

		// expected values
		home, kconfig string
		debug         bool
	}{
		{
			name:    "defaults",
			args:    []string{},
			home:    DefaultKudoHome,
			kconfig: os.Getenv("HOME") + "/.kube/config",
		},
		{
			name:    "with flags set",
			args:    []string{"--home", "/foo", "--debug", "--kubeconfig", "/bar"},
			home:    "/foo",
			kconfig: "/bar",
			debug:   true,
		},
		{
			name:    "with ENV set",
			args:    []string{},
			envars:  map[string]string{"KUDO_HOME": "/bar", "KUDO_DEBUG": "1", "KUBECONFIG": "/foo"},
			home:    "/bar",
			debug:   true,
			kconfig: "/foo",
		},
		{
			name:    "with flags and ENV set",
			args:    []string{"--home", "/foo", "--debug", "--kubeconfig", "/bar"},
			envars:  map[string]string{"KUDO_HOME": "/bar", "KUDO_DEBUG": "1", "KUBECONFIG": "/foo"},
			home:    "/foo",
			debug:   true,
			kconfig: "/bar",
		},
	}

	allEnvvars := map[string]string{
		"KUDO_DEBUG": "",
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
			if settings.Debug != tt.debug {
				t.Errorf("expected debug %t, got %t", tt.debug, settings.Debug)
			}
			if settings.KubeConfig != tt.kconfig {
				t.Errorf("expected kubeconfig %q, got %q", tt.kconfig, settings.KubeConfig)
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
