package diagnostics

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"

	"github.com/spf13/afero"
)

const (
	appKudoManager = "kudo-manager"
)

const (
	continueOnError = false
	failOnError     = true
)

type Options struct {
	Instance string
	LogSince int64
}

type Collector interface {
	Collect() error
}

func Collect(fs afero.Fs, options *Options, s *env.Settings) error {
	ir, err := NewInstanceResources(options, s)
	if err != nil {
		return err
	}
	instanceDiagRunner := &Runner{}
	ctx := &processingContext{root: DiagDir, instanceName: options.Instance}
	p := &ObjectPrinter{fs: fs}

	instanceDiagRunner.
		Run(ResourceCollectorGroup{
			{ir.Instance, "instance", ctx.attachToOperator, failOnError, ctx.mustSetOperatorVersionNameFromInstance, p, ObjectWithDir},
			{ir.OperatorVersion(ctx.operatorVersionName), "operatorversion", ctx.attachToOperator, failOnError, ctx.mustSetOperatorNameFromOperatorVersion, p, ObjectWithDir},
			{ir.Operator(ctx.operatorName), "operator", ctx.attachToRoot, failOnError, nil, p, ObjectWithDir}}).
		Run(&ResourceCollector{ir.Pods, "pod", ctx.attachToInstance, continueOnError, ctx.mustAddPodNames, p, ObjectListWithDirs}).
		Run(&ResourceCollector{ir.Services, "service", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.Deployments, "deployment", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.ReplicaSets, "replicaset", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.StatefulSets, "statefulset", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.ServiceAccounts, "serviceaccount", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.ClusterRoleBindings, "clusterrolebinding", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.RoleBindings, "rolebinding", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.ClusterRoles, "clusterrole", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{ir.Roles, "role", ctx.attachToInstance, continueOnError, nil, p, RuntimeObject}).
		RunForEach(ctx.podNames, func(podName string) Collector { return &LogCollector{ir, podName, ctx.attachToInstance, p} }).
		DumpToYaml(version.Get(), ctx.attachToRoot, "version", p).
		DumpToYaml(s, ctx.attachToRoot, "settings", p)

	kr, err := NewKudoResources(s)
	if err != nil {
		return err
	}

	ctx = &processingContext{root: KudoDir}
	kudoDiagRunner := &Runner{}
	kudoDiagRunner.
		Run(&ResourceCollector{kr.Pods, "pod", ctx.attachToRoot, continueOnError, ctx.mustAddPodNames, p, ObjectListWithDirs}).
		Run(&ResourceCollector{kr.Services, "service", ctx.attachToRoot, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{kr.StatefulSets, "statefulset", ctx.attachToRoot, continueOnError, nil, p, RuntimeObject}).
		Run(&ResourceCollector{kr.ServiceAccounts, "serviceaccount", ctx.attachToRoot, continueOnError, nil, p, RuntimeObject}).
		RunForEach(ctx.podNames, func(podName string) Collector { return &LogCollector{kr, podName, ctx.attachToRoot, p} })

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
