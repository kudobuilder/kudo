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
	rh := runnerHelper{p}

	instanceErr := rh.runForInstance(instance, options, c, version.Get(), s)
	kudoErr := rh.runForKudoManager(options, c)

	errMsgs := p.errors
	if instanceErr != nil {
		errMsgs = append(errMsgs, instanceErr.Error())
	}
	if kudoErr != nil {
		errMsgs = append(errMsgs, kudoErr.Error())
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "\n"))
	}
	return nil
}
