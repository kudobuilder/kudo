package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

type runnerHelper struct {
	p *nonFailingPrinter
}

func (rh *runnerHelper) runForInstance(instance string, options *Options, c *kudo.Client, info version.Info, s *env.Settings) error {
	ir, err := newInstanceResources(instance, options, c, s)
	if err != nil {
		return err
	}

	ctx := &processingContext{root: DiagDir, instanceName: instance}
	instanceDiagRunner := &runner{}
	instanceDiagRunner.
		run(resourceCollectorGroup{
			{
				loadResourceFn: ir.instance,
				errKind:        "instance",
				parentDir:      ctx.operatorDirectory,
				failOnError:    true,
				callback:       ctx.mustSetOperatorVersionNameFromInstance,
				printer:        rh.p,
				printMode:      ObjectWithDir},
			{
				loadResourceFn: ir.operatorVersion(ctx.operatorVersionName),
				errKind:        "operatorversion",
				parentDir:      ctx.operatorDirectory,
				failOnError:    true,
				callback:       ctx.mustSetOperatorNameFromOperatorVersion,
				printer:        rh.p,
				printMode:      ObjectWithDir},
			{
				loadResourceFn: ir.operator(ctx.operatorName),
				errKind:        "operator",
				parentDir:      ctx.rootDirectory,
				failOnError:    true,
				printer:        rh.p,
				printMode:      ObjectWithDir}}).
		run(&resourceCollector{
			loadResourceFn: ir.pods,
			errKind:        "pod",
			parentDir:      ctx.instanceDirectory,
			callback:       ctx.mustAddPodNames,
			printer:        rh.p,
			printMode:      ObjectListWithDirs}).
		run(&resourceCollector{
			loadResourceFn: ir.services,
			errKind:        "service",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.deployments,
			errKind:        "deployment",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.replicaSets,
			errKind:        "replicaset",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.statefulSets,
			errKind:        "statefulset",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.serviceAccounts,
			errKind:        "serviceaccount",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.clusterRoleBindings,
			errKind:        "clusterrolebinding",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.roleBindings,
			errKind:        "rolebinding",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.clusterRoles,
			errKind:        "clusterrole",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.roles,
			errKind:        "role",
			parentDir:      ctx.instanceDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		runForEach(ctx.podNames,
			func(podName string) collector {
				return &logCollector{loadLogFn: ir.log,
					podName:   podName,
					parentDir: ctx.instanceDirectory,
					printer:   rh.p}
			}).
		dumpToYaml(info, ctx.rootDirectory, "version", rh.p).
		dumpToYaml(s, ctx.rootDirectory, "settings", rh.p)

	return instanceDiagRunner.fatalErr
}

func (rh *runnerHelper) runForKudoManager(options *Options, c *kudo.Client) error {
	kr, err := newKudoResources(options, c)
	if err != nil {
		return err
	}
	ctx := &processingContext{root: KudoDir}
	kudoDiagRunner := &runner{}
	kudoDiagRunner.
		run(&resourceCollector{
			loadResourceFn: kr.pods,
			errKind:        "pod",
			parentDir:      ctx.rootDirectory,
			callback:       ctx.mustAddPodNames,
			printer:        rh.p,
			printMode:      ObjectListWithDirs}).
		run(&resourceCollector{
			loadResourceFn: kr.services,
			errKind:        "service",
			parentDir:      ctx.rootDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: kr.statefulSets,
			errKind:        "statefulset",
			parentDir:      ctx.rootDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: kr.serviceAccounts,
			errKind:        "serviceaccount",
			parentDir:      ctx.rootDirectory,
			printer:        rh.p,
			printMode:      RuntimeObject}).
		runForEach(ctx.podNames, func(podName string) collector {
			return &logCollector{loadLogFn: kr.log, podName: podName, parentDir: ctx.rootDirectory, printer: rh.p}
		})
	return kudoDiagRunner.fatalErr
}
