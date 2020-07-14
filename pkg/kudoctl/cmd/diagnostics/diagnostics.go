package diagnostics

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

type Options struct {
	LogSince *int64
}

func NewOptions(logSince time.Duration) *Options {
	opts := Options{}
	if logSince > 0 {
		sec := int64(logSince.Round(time.Second).Seconds())
		opts.LogSince = &sec
	}
	return &opts
}

func Collect(fs afero.Fs, instance, directory string, options *Options, c *kudo.Client, s *env.Settings) error {
	if err := verifyDiagDirNotExists(fs, directory); err != nil {
		return err
	}
	p := &nonFailingPrinter{fs: fs}

	if err := diagForInstance(instance, directory, options, c, version.Get(), s, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if err := diagForKudoManager(options, directory, c, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}

func verifyDiagDirNotExists(fs afero.Fs, directory string) error {
	exists, err := afero.Exists(fs, directory)
	if err != nil {
		return fmt.Errorf("failed to verify that target directory %s doesn't exist: %v", directory, err)
	}
	if exists {
		return fmt.Errorf("target directory %s already exists", directory)
	}
	return nil
}
