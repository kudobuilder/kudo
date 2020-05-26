package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

func runForInstance(instance string, options *Options, c *kudo.Client, info version.Info, s *env.Settings, p *nonFailingPrinter) error {
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
				name:           "instance",
				parentDir:      ctx.operatorDirectory,
				failOnError:    true,
				callback:       ctx.mustSetOperatorVersionNameFromInstance,
				printer:        p,
				printMode:      ObjectWithDir},
			{
				loadResourceFn: ir.operatorVersion(ctx.operatorVersionName),
				name:           "operatorversion",
				parentDir:      ctx.operatorDirectory,
				failOnError:    true,
				callback:       ctx.mustSetOperatorNameFromOperatorVersion,
				printer:        p,
				printMode:      ObjectWithDir},
			{
				loadResourceFn: ir.operator(ctx.operatorName),
				name:           "operator",
				parentDir:      ctx.rootDirectory,
				failOnError:    true,
				printer:        p,
				printMode:      ObjectWithDir}}).
		run(&resourceCollector{
			loadResourceFn: ir.pods,
			name:           "pod",
			parentDir:      ctx.instanceDirectory,
			callback:       ctx.mustAddPodNames,
			printer:        p,
			printMode:      ObjectListWithDirs}).
		run(&resourceCollector{
			loadResourceFn: ir.services,
			name:           "service",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.deployments,
			name:           "deployment",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.replicaSets,
			name:           "replicaset",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.statefulSets,
			name:           "statefulset",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.serviceAccounts,
			name:           "serviceaccount",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.clusterRoleBindings,
			name:           "clusterrolebinding",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.roleBindings,
			name:           "rolebinding",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.clusterRoles,
			name:           "clusterrole",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: ir.roles,
			name:           "role",
			parentDir:      ctx.instanceDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		runForEach(ctx.podNames,
			func(podName string) collector {
				return &logCollector{loadLogFn: ir.log,
					podName:   podName,
					parentDir: ctx.instanceDirectory,
					printer:   p}
			}).
		dumpToYaml(info, ctx.rootDirectory, "version", p).
		dumpToYaml(s, ctx.rootDirectory, "settings", p)

	return instanceDiagRunner.fatalErr
}

func runForKudoManager(options *Options, c *kudo.Client, p *nonFailingPrinter) error {
	kr, err := newKudoResources(options, c)
	if err != nil {
		return err
	}
	ctx := &processingContext{root: KudoDir}
	kudoDiagRunner := &runner{}
	kudoDiagRunner.
		run(&resourceCollector{
			loadResourceFn: kr.pods,
			name:           "pod",
			parentDir:      ctx.rootDirectory,
			callback:       ctx.mustAddPodNames,
			printer:        p,
			printMode:      ObjectListWithDirs}).
		run(&resourceCollector{
			loadResourceFn: kr.services,
			name:           "service",
			parentDir:      ctx.rootDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: kr.statefulSets,
			name:           "statefulset",
			parentDir:      ctx.rootDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		run(&resourceCollector{
			loadResourceFn: kr.serviceAccounts,
			name:           "serviceaccount",
			parentDir:      ctx.rootDirectory,
			printer:        p,
			printMode:      RuntimeObject}).
		runForEach(ctx.podNames, func(podName string) collector {
			return &logCollector{loadLogFn: kr.log, podName: podName, parentDir: ctx.rootDirectory, printer: p}
		})
	return kudoDiagRunner.fatalErr
}
