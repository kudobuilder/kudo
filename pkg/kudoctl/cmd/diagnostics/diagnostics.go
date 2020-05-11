package diagnostics

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	nsKudoSystem      = "kudo-system"
	labelKudoOperator = "kudo.dev/operator"
	appKudoManager    = "kudo-manager"
)

type Options struct {
	Instance string
	LogSince int64
}

type collector interface {
	Collect() (Printable, error)
}

type compositeCollector struct {
	collectors []collector
}

// TODO: add an option of not failing on error
func (cc *compositeCollector) Collect() (Printable, error) {
	l := len(cc.collectors)
	ret := make([]Printable, l)
	for i := 0; i < len(cc.collectors); i++ {
		p, err := cc.collectors[i].Collect()
		if err != nil {
			return nil, err // TODO: need "Printable error" type
		}
		ret[l-1-i] = p
	}
	return PrintableList(ret), nil
}

func Collect(fs afero.Fs, options *Options, s *env.Settings) error {
	ir, err := NewInstanceResources(options, s)
	if err != nil {
		return err
	}

	err = (&SimpleBuilder{
		r:  ir,
		fs: fs,
	}).
		AddGroup(
			ResourceWithContext(attachToOperator, ObjectWithDir, Instance),
			ResourceWithContext(attachToOperator, ObjectWithDir, OperatorVersion),
			ResourceWithContext(attachToRoot, ObjectWithDir, Operator)).
		Add(ResourceWithContext(attachToInstance, ObjectsWithDir, Pods)).
		Add(Resource(attachToInstance, RuntimeObject, Services)).
		Add(Resource(attachToInstance, RuntimeObject, Deployments)).
		Add(Resource(attachToInstance, RuntimeObject, ReplicaSets)).
		Add(Resource(attachToInstance, RuntimeObject, StatefulSets)).
		Add(Resource(attachToInstance, RuntimeObject, ServiceAccounts)).
		Add(Resource(attachToInstance, RuntimeObject, ClusterRoleBindings)).
		Add(Resource(attachToInstance, RuntimeObject, RoleBindings)).
		Add(Resource(attachToInstance, RuntimeObject, ClusterRoles)).
		Add(Resource(attachToInstance, RuntimeObject, Roles)).
		Add(Logs(attachToInstance)).
		AddAsYaml(attachToRoot, "version", version.Get()).
		AddAsYaml(attachToRoot, "settings", s).
		Run()

	if err != nil {
		return err
	}

	kr, err := NewKudoResources(s)
	if err != nil {
		return err
	}

	err = (&SimpleBuilder{
		r:  kr,
		fs: fs,
	}).
		Add(ResourceWithContext(attachToKudoRoot, ObjectsWithDir, Pods)).
		Add(Resource(attachToKudoRoot, RuntimeObject, Services)).
		Add(Resource(attachToKudoRoot, RuntimeObject, StatefulSets)).
		Add(Resource(attachToKudoRoot, RuntimeObject, ServiceAccounts)).
		Add(Logs(attachToKudoRoot)).
		Run()

	return err
}

func ResourceWithContext(baseDir func(*processingContext) string, mode printMode, r ResourceFnWithContext) func(*SimpleBuilder) collector {
	return func(b *SimpleBuilder) collector {
		return b.createResourceCollector(r.toResourceFn(&b.ctx), baseDir, mode)
	}
}

func Resource(baseDir func(*processingContext) string, mode printMode, r ResourceFn) func(*SimpleBuilder) collector {
	return func(b *SimpleBuilder) collector {
		return b.createResourceCollector(r, baseDir, mode)
	}
}

func Logs(baseDir func(*processingContext) string) func(*SimpleBuilder) collector {
	return func(b *SimpleBuilder) collector {
		var ret []collector
		ret = append(ret, &LogCollector{
			r:        b.r,
			podNames: func() []string { return b.ctx.podNames },
			baseDir:  func() string { return baseDir(&b.ctx) },
		})
		return &compositeCollector{ret}
	}
}
