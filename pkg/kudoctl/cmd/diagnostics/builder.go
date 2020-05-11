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

type SimpleBuilder struct {
	r          *ResourceFuncsConfig
	ctx        processingContext
	collectors []collector
	fs         afero.Fs
}

func (b *SimpleBuilder) Collect(baseDir func(*processingContext) string, asObject printMode, r ResourceFn) *SimpleBuilder {
	b.collectors = append(b.collectors, b.createResourceCollector(r, baseDir, asObject))
	return b
}

func (b *SimpleBuilder) AddGroup(fns ...func(*SimpleBuilder) collector) *SimpleBuilder {
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

func (b *SimpleBuilder) Add(fn func(*SimpleBuilder) collector) *SimpleBuilder {
	b.collectors = append(b.collectors, fn(b))
	return b
}

func (b *SimpleBuilder) AddAsYaml(baseDir func(ctx *processingContext) string, name string, o interface{}) *SimpleBuilder {
	b.collectors = append(b.collectors, &AnyPrintable{
		dir:  func() string { return baseDir(&b.ctx) },
		name: name,
		v:    o,
	})
	return b
}

func (b *SimpleBuilder) Run() error {
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

func (b *SimpleBuilder) createResourceCollector(fn ResourceFn, baseDir func(*processingContext) string, mode printMode) *ResourceCollector {
	return &ResourceCollector{
		r:          b.r,
		resourceFn: fn,
		printMode:  mode,
		baseDir:    func() string { return baseDir(&b.ctx) },
	}
}
