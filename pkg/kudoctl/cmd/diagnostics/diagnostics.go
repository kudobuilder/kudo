package diagnostics

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

const DefaultDiagDir = "diag"

type Options struct {
	LogSince  *int64
	outputDir string
}

func (o *Options) DiagDir() string {
	return o.outputDir
}

func (o *Options) KudoDir() string {
	return path.Join(o.DiagDir(), "kudo")
}

func NewDefaultOptions() *Options {
	return &Options{
		LogSince:  nil,
		outputDir: DefaultDiagDir,
	}
}

func NewOptions(logSince time.Duration, outputDir string) *Options {
	opts := NewDefaultOptions()
	if logSince > 0 {
		sec := int64(logSince.Round(time.Second).Seconds())
		opts.LogSince = &sec
	}
	if outputDir != "" {
		opts.outputDir = outputDir
	}
	return opts
}

func Collect(fs afero.Fs, instance string, options *Options, c *kudo.Client, s *env.Settings) error {
	if err := verifyDiagDirNotExists(fs, options); err != nil {
		return err
	}
	p := &nonFailingPrinter{fs: fs}

	if err := diagForInstance(instance, options, c, version.Get(), s, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if err := diagForKudoManager(options, c, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}

func verifyDiagDirNotExists(fs afero.Fs, options *Options) error {
	exists, err := afero.Exists(fs, options.DiagDir())
	if err != nil {
		return fmt.Errorf("failed to verify that target directory %s doesn't exist: %v", options.DiagDir(), err)
	}
	if exists {
		return fmt.Errorf("target directory %s already exists", options.DiagDir())
	}
	return nil
}
