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

func Collect(fs afero.Fs, instance string, options *Options, c *kudo.Client, s *env.Settings) error {
	p := &nonFailingPrinter{fs: fs}

	if err := runForInstance(instance, options, c, version.Get(), s, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if err := runForKudoManager(options, c, p); err != nil {
		p.errors = append(p.errors, err.Error())
	}
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}
