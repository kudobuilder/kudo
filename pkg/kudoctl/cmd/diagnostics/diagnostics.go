package diagnostics

import (
	"fmt"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"

	"github.com/spf13/afero"
)

type Options struct {
	Instance string
	LogSince int64
}

func NewOptions(instance string, logSince time.Duration) *Options {
	opts := Options{Instance: instance}
	if logSince > 0 {
		sec := int64(logSince.Round(time.Second).Seconds())
		opts.LogSince = sec
	}
	return &opts
}

func Collect(fs afero.Fs, options *Options, c *kudo.Client, s *env.Settings) error {
	ir, err := NewInstanceResources(options, c, s)
	if err != nil {
		return err
	}
	instanceDiagRunner := &Runner{}
	ctx := &processingContext{root: DiagDir, instanceName: options.Instance}
	p := &NonFailingPrinter{fs: fs}

	instanceDiagRunner.
		Run(&InstanceCollector{fs, options, c, s, ctx, p, ir}).
		DumpToYaml(version.Get(), ctx.attachToRoot, "version", p).
		DumpToYaml(s, ctx.attachToRoot, "settings", p)

	kr, err := NewKudoResources(c)
	if err != nil {
		return err
	}
	kudoDiagRunner := &Runner{}
	ctx = &processingContext{root: KudoDir}
	kudoDiagRunner.Run(&KudoManagerCollector{fs, options, c, s, ctx, p, kr})

	errMsgs := p.errors
	if instanceDiagRunner.fatalErr != nil {
		errMsgs = append(errMsgs, instanceDiagRunner.fatalErr.Error())
	}
	if kudoDiagRunner.fatalErr != nil {
		errMsgs = append(errMsgs, instanceDiagRunner.fatalErr.Error())
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "\n"))
	}
	return nil
}
