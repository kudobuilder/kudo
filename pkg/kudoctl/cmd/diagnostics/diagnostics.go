package diagnostics

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	appKudoManager    = "kudo-manager"
)

const (
	continueOnError = false
	failOnError     = true
)

type Options struct {
	Instance string
	LogSince int64
}

// collector - implementors are expected to return error only if fatal, otherwise the error should be wrapped and
// returned in place of Printable to print it instead of the object that failed to be collected
type collector interface {
	Collect() (Printable, error)
}

// compositeCollector - auxiliary union of collectors related via their side-effects on processingContext
type compositeCollector struct {
	collectors []collector
}

// Collect - flow ensures that objects are printed if and only after all of them are collected
func (cc *compositeCollector) Collect() (Printable, error) {
	l := len(cc.collectors)
	ret := make([]Printable, l)
	for i := 0; i < len(cc.collectors); i++ {
		p, err := cc.collectors[i].Collect()
		if err != nil {
			return nil, err // if a collector returns error it's fatal
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

	err = (&Builder{
		r:  ir,
		fs: fs,
	}).
		AddGroup(
			ResourceWithContext(attachToOperator, ObjectWithDir, Instance, "instance", failOnError),
			ResourceWithContext(attachToOperator, ObjectWithDir, OperatorVersion, "operatorversion", failOnError),
			ResourceWithContext(attachToRoot, ObjectWithDir, Operator, "operator", failOnError)).
		Add(ResourceWithContext(attachToInstance, ObjectListWithDirs, Pods, "pods", failOnError)).
		Add(Resource(attachToInstance, RuntimeObject, Services, "services", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, Deployments, "deployments", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, ReplicaSets, "replicasets", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, StatefulSets, "statefulsets", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, ServiceAccounts, "serviceaccounts", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, ClusterRoleBindings, "clusterrolebindings", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, RoleBindings, "rolebindings", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, ClusterRoles, "clusterroles", continueOnError)).
		Add(Resource(attachToInstance, RuntimeObject, Roles, "roles", continueOnError)).
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

	err = (&Builder{
		r:  kr,
		fs: fs,
	}).
		Add(ResourceWithContext(attachToKudoRoot, ObjectListWithDirs, Pods, "pods", failOnError)).
		Add(Resource(attachToKudoRoot, RuntimeObject, Services, "services", continueOnError)).
		Add(Resource(attachToKudoRoot, RuntimeObject, StatefulSets, "statefulsets", continueOnError)).
		Add(Resource(attachToKudoRoot, RuntimeObject, ServiceAccounts, "serviceaccounts", continueOnError)).
		Add(Logs(attachToKudoRoot)).
		Run()

	return err
}

func ResourceWithContext(baseDir func(*processingContext) string, mode printMode, r ResourceFnWithContext, errName string, failOnErr bool) func(*Builder) collector {
	return func(b *Builder) collector {
		return b.createResourceCollector(r.toResourceFn(&b.ctx), baseDir, mode, errName, failOnErr)
	}
}

func Resource(baseDir func(*processingContext) string, mode printMode, r ResourceFn, errName string, failOnErr bool) func(*Builder) collector {
	return func(b *Builder) collector {
		return b.createResourceCollector(r, baseDir, mode, errName, failOnErr)
	}
}

func Logs(baseDir func(*processingContext) string) func(*Builder) collector {
	return func(b *Builder) collector {
		return &LogCollector{
			r:         b.r,
			podNames:  func() []string { return b.ctx.podNames },
			parentDir: func() string { return baseDir(&b.ctx) },
		}
	}
}
