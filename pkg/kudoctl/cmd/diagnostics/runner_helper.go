package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

func diagForInstance(instance string, options *Options, c *kudo.Client, info version.Info, s *env.Settings, p *nonFailingPrinter) error {
	ir, err := newInstanceResources(instance, options, c, s)
	if err != nil {
		p.printError(err, options.DiagDir(), "instance")
		return err
	}

	ctx := &processingContext{root: options.DiagDir(), instanceName: instance}

	runner := runnerForInstance(ir, ctx)
	runner.addObjDump(info, ctx.rootDirectory, "version")
	runner.addObjDump(s, ctx.rootDirectory, "settings")

	if err := runner.run(p); err != nil {
		return err
	}

	deps, err := newDependenciesResources(instance, options, c, s)
	if err != nil {
		p.printError(err, options.DiagDir(), "instance")
		return err
	}

	for _, dep := range deps {
		// Nest the dependencies in the parents operator directory
		root := ctx.operatorDirectory()

		depCtx := &processingContext{root: root, instanceName: dep.instanceObj.Name}

		runner := runnerForInstance(dep, depCtx)
		if err := runner.run(p); err != nil {
			return err
		}
	}

	return nil
}

func diagForKudoManager(options *Options, c *kudo.Client, p *nonFailingPrinter) error {
	kr, err := newKudoResources(options, c)
	if err != nil {
		return err
	}
	ctx := &processingContext{root: options.KudoDir()}

	runner := runnerForKudoManager(kr, ctx)

	if err := runner.run(p); err != nil {
		return err
	}

	return nil
}

func runnerForInstance(ir *resourceFuncsConfig, ctx *processingContext) *runner {
	r := &runner{}

	instance := resourceCollectorGroup{[]resourceCollector{
		{
			loadResourceFn: ir.instance,
			name:           "instance",
			parentDir:      ctx.operatorDirectory,
			failOnError:    true,
			callback:       ctx.setOperatorVersionNameFromInstance,
			printMode:      ObjectWithDir},
		{
			loadResourceFn: ir.operatorVersion(ctx.operatorVersionName),
			name:           "operatorversion",
			parentDir:      ctx.operatorDirectory,
			failOnError:    true,
			callback:       ctx.setOperatorNameFromOperatorVersion,
			printMode:      ObjectWithDir},
		{
			loadResourceFn: ir.operator(ctx.operatorName),
			name:           "operator",
			parentDir:      ctx.rootDirectory,
			failOnError:    true,
			printMode:      ObjectWithDir},
	}}
	r.addCollector(instance)

	r.addCollector(&resourceCollector{
		loadResourceFn: ir.pods,
		name:           "pod",
		parentDir:      ctx.instanceDirectory,
		callback:       ctx.setPods,
		printMode:      ObjectListWithDirs})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.services,
		name:           "service",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.deployments,
		name:           "deployment",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.statefulSets,
		name:           "statefulset",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.replicaSets,
		name:           "replicaset",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.statefulSets,
		name:           "statefulset",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.serviceAccounts,
		name:           "serviceaccount",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.clusterRoleBindings,
		name:           "clusterrolebinding",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.roleBindings,
		name:           "rolebinding",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.clusterRoles,
		name:           "clusterrole",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: ir.roles,
		name:           "role",
		parentDir:      ctx.instanceDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&logsCollector{
		loadLogFn: ir.log,
		pods:      ctx.podList,
		parentDir: ctx.instanceDirectory,
	})

	return r
}

func runnerForKudoManager(kr *resourceFuncsConfig, ctx *processingContext) *runner {
	r := &runner{}

	r.addCollector(&resourceCollector{
		loadResourceFn: kr.pods,
		name:           "pod",
		parentDir:      ctx.rootDirectory,
		callback:       ctx.setPods,
		printMode:      ObjectListWithDirs})
	r.addCollector(&resourceCollector{
		loadResourceFn: kr.services,
		name:           "service",
		parentDir:      ctx.rootDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: kr.statefulSets,
		name:           "statefulset",
		parentDir:      ctx.rootDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&resourceCollector{
		loadResourceFn: kr.serviceAccounts,
		name:           "serviceaccount",
		parentDir:      ctx.rootDirectory,
		printMode:      RuntimeObject})
	r.addCollector(&logsCollector{
		loadLogFn: kr.log,
		pods:      ctx.podList,
		parentDir: ctx.rootDirectory})

	return r
}
