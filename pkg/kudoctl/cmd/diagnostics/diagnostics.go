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
		Run(ResourceCollectorGroup{
			{
				loadResourceFn: ir.Instance,
				errKind:        "instance",
				parentDir:      ctx.attachToOperator,
				failOnError:    true,
				callback:       ctx.mustSetOperatorVersionNameFromInstance,
				printer:        p, printMode: ObjectWithDir},
			{
				loadResourceFn: ir.OperatorVersion(ctx.operatorVersionName),
				errKind:        "operatorversion", parentDir: ctx.attachToOperator,
				failOnError: true, callback: ctx.mustSetOperatorNameFromOperatorVersion,
				printer: p, printMode: ObjectWithDir},
			{
				loadResourceFn: ir.Operator(ctx.operatorName),
				errKind:        "operator",
				parentDir:      ctx.attachToRoot,
				failOnError:    true,
				printer:        p,
				printMode:      ObjectWithDir}}).
		Run(&ResourceCollector{
			loadResourceFn: ir.Pods, errKind: "pod",
			parentDir: ctx.attachToInstance,
			callback:  ctx.mustAddPodNames,
			printer:   p,
			printMode: ObjectListWithDirs}).
		Run(&ResourceCollector{
			loadResourceFn: ir.Services,
			errKind:        "service",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.Deployments,
			errKind:        "deployment",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.ReplicaSets,
			errKind:        "replicaset",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.StatefulSets,
			errKind:        "statefulset",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.ServiceAccounts,
			errKind:        "serviceaccount",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.ClusterRoleBindings,
			errKind:        "clusterrolebinding",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.RoleBindings,
			errKind:        "rolebinding",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.ClusterRoles,
			errKind:        "clusterrole",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: ir.Roles,
			errKind:        "role",
			parentDir:      ctx.attachToInstance,
			printer:        p,
			printMode:      RuntimeObject}).
		RunForEach(ctx.podNames,
			func(podName string) Collector {
				return &LogCollector{loadLogFn: ir.Log,
					podName:   podName,
					parentDir: ctx.attachToInstance,
					printer:   p}
			}).
		DumpToYaml(version.Get(), ctx.attachToRoot, "version", p).
		DumpToYaml(s, ctx.attachToRoot, "settings", p)

	kr, err := NewKudoResources(c)
	if err != nil {
		return err
	}

	ctx = &processingContext{root: KudoDir}
	kudoDiagRunner := &Runner{}
	kudoDiagRunner.
		Run(&ResourceCollector{
			loadResourceFn: kr.Pods,
			errKind:        "pod",
			parentDir:      ctx.attachToRoot,
			callback:       ctx.mustAddPodNames,
			printer:        p,
			printMode:      ObjectListWithDirs}).
		Run(&ResourceCollector{
			loadResourceFn: kr.Services,
			errKind:        "service",
			parentDir:      ctx.attachToRoot,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: kr.StatefulSets,
			errKind:        "statefulset",
			parentDir:      ctx.attachToRoot,
			printer:        p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: kr.ServiceAccounts,
			errKind:        "serviceaccount",
			parentDir:      ctx.attachToRoot,
			printer:        p,
			printMode:      RuntimeObject}).
		RunForEach(ctx.podNames, func(podName string) Collector {
			return &LogCollector{loadLogFn: kr.Log, podName: podName, parentDir: ctx.attachToRoot, printer: p}
		})

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
