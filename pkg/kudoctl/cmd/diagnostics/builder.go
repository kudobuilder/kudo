package diagnostics

import (
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceFn func(*ResourceFuncsConfig) (runtime.Object, error)
type ResourceFnWithContext func(*ResourceFuncsConfig, *processingContext) (runtime.Object, error)

func (fn ResourceFnWithContext) toResourceFn(ctx *processingContext) ResourceFn {
	return func(r *ResourceFuncsConfig) (runtime.Object, error) {
		return fn(r, ctx)
	}
}

type Builder struct {
	r          *ResourceFuncsConfig
	ctx        processingContext
	collectors []collector
	fs         afero.Fs
}

func (b *Builder) AddGroup(fns ...func(*Builder) collector) *Builder {
	if len(fns) == 0 {
		return b
	}
	var collectors []collector
	for _, fn := range fns {
		collectors = append(collectors, fn(b))
	}
	b.collectors = append(b.collectors, &compositeCollector{collectors: collectors})
	return b
}

func (b *Builder) Add(fn func(*Builder) collector) *Builder {
	b.collectors = append(b.collectors, fn(b))
	return b
}

func (b *Builder) AddAsYaml(baseDir func(ctx *processingContext) string, name string, o interface{}) *Builder {
	b.collectors = append(b.collectors, &PrintableYaml{
		dir:  func() string { return baseDir(&b.ctx) },
		name: name,
		v:    o,
	})
	return b
}

func (b *Builder) Run() error {
	for _, c := range b.collectors {
		p, err := c.Collect()
		if err != nil {
			return err
		}
		if p == nil {
			continue
		}
		err = p.print(b.fs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) createResourceCollector(fn ResourceFn, baseDir func(*processingContext) string, mode printMode, errName string, failOnErr bool) *ResourceCollector {
	return &ResourceCollector{
		r:          b.r,
		resourceFn: fn,
		printMode:  mode,
		parentDir:  func() string { return baseDir(&b.ctx) },
		errName:    errName,
		failOnErr:  failOnErr,
	}
}
