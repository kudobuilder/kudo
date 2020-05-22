package diagnostics

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/spf13/afero"
	"io"
	"path/filepath"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceCollector - Collector interface implementation for Kubernetes resources (runtime objects)
type ResourceCollector struct {
	loadResourceFn func() (runtime.Object, error)
	errKind        string                 // object kind used to describe the error
	parentDir      func() string          // parent dir to attach the printer's output
	failOnError    bool                   // define whether the collector should return the error
	callback       func(o runtime.Object) // should be used to update some shared context
	printer        *NonFailingPrinter
	printMode      printMode
}

// Collect - load a resource and send either the resource or collection error to printer
// return error if failOnError field is set to true
// if failOnError is true, finding no object(s) is treated as an error
func (c *ResourceCollector) Collect() error {
	obj, err := c.loadResourceFn()
	switch {
	case err != nil:
		if c.failOnError {
			return fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.errKind, err)
		}
		c.printer.printError(err, c.parentDir(), c.errKind)
	case obj == nil || reflect.ValueOf(obj).IsNil() || meta.IsListType(obj) && meta.LenList(obj) == 0:
		if c.failOnError {
			return fmt.Errorf("no object(s) of kind %s retrieved", c.errKind)
		}
	default:
		if c.callback != nil {
			c.callback(obj)
		}
		c.printer.printObject(obj, c.parentDir(), c.printMode)
	}
	return nil
}

// ResourceCollectorGroup - a composite collector for Kubernetes runtime objects whose loading and printing depend on
// each other's side-effects on the shared context
type ResourceCollectorGroup []ResourceCollector

// Collect - collect resource and run callback for each collector, print all afterwards
// collection failures are treated as fatal regardless of the collectors failOnError flag setting
func (g ResourceCollectorGroup) Collect() error {
	objs := make([]runtime.Object, len(g))
	modes := make([]printMode, len(g))
	for i, c := range g {
		obj, err := c.loadResourceFn()
		if err != nil {
			return fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.errKind, err)
		}
		if obj == nil || reflect.ValueOf(obj).IsNil() || meta.IsListType(obj) && meta.LenList(obj) == 0 {
			return fmt.Errorf("no object(s) of kind %s retrieved", c.errKind)
		}
		if c.callback != nil {
			c.callback(obj)
		}
		objs[i] = obj
	}
	for i, c := range g {
		c.printer.printObject(objs[i], c.parentDir(), modes[i])
	}
	return nil
}

// LogCollector - Collector interface implementation for logs for a pod
type LogCollector struct {
	loadLogFn func(string) (io.ReadCloser, error)
	podName   string
	parentDir func() string
	printer   *NonFailingPrinter
}

// Collect - prints either a log or an error. Always returns nil error.
func (c *LogCollector) Collect() error {
	log, err := c.loadLogFn(c.podName)
	if err != nil {
		c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", c.podName)), fmt.Sprintf("%s.log", c.podName))
	} else {
		c.printer.printLog(log, c.parentDir(), c.podName)
		_ = log.Close()
	}
	return nil
}

type InstanceCollector struct {
	fs      afero.Fs
	options *Options
	c       *kudo.Client
	s       *env.Settings
	ctx     *processingContext
	p       *NonFailingPrinter
	ir      *ResourceFuncsConfig
}

func (c *InstanceCollector) Collect() error {

	instanceDiagRunner := &Runner{}

	instanceDiagRunner.
		Run(ResourceCollectorGroup{
			{
				loadResourceFn: c.ir.Instance,
				errKind:        "instance",
				parentDir:      c.ctx.attachToOperator,
				failOnError:    true,
				callback:       c.ctx.mustSetOperatorVersionNameFromInstance,
				printer:        c.p, printMode: ObjectWithDir},
			{
				loadResourceFn: c.ir.OperatorVersion(c.ctx.operatorVersionName),
				errKind:        "operatorversion", parentDir: c.ctx.attachToOperator,
				failOnError: true, callback: c.ctx.mustSetOperatorNameFromOperatorVersion,
				printer: c.p, printMode: ObjectWithDir},
			{
				loadResourceFn: c.ir.Operator(c.ctx.operatorName),
				errKind:        "operator",
				parentDir:      c.ctx.attachToRoot,
				failOnError:    true,
				printer:        c.p,
				printMode:      ObjectWithDir}}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.Pods, errKind: "pod",
			parentDir: c.ctx.attachToInstance,
			callback:  c.ctx.mustAddPodNames,
			printer:   c.p,
			printMode: ObjectListWithDirs}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.Services,
			errKind:        "service",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.Deployments,
			errKind:        "deployment",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.ReplicaSets,
			errKind:        "replicaset",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.StatefulSets,
			errKind:        "statefulset",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.ServiceAccounts,
			errKind:        "serviceaccount",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.ClusterRoleBindings,
			errKind:        "clusterrolebinding",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.RoleBindings,
			errKind:        "rolebinding",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.ClusterRoles,
			errKind:        "clusterrole",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.ir.Roles,
			errKind:        "role",
			parentDir:      c.ctx.attachToInstance,
			printer:        c.p,
			printMode:      RuntimeObject}).
		RunForEach(c.ctx.podNames,
			func(podName string) Collector {
				return &LogCollector{loadLogFn: c.ir.Log,
					podName:   podName,
					parentDir: c.ctx.attachToInstance,
					printer:   c.p}
			})
	return instanceDiagRunner.fatalErr
}

type KudoManagerCollector struct {
	fs      afero.Fs
	options *Options
	c       *kudo.Client
	s       *env.Settings
	ctx     *processingContext
	p       *NonFailingPrinter
	kr      *ResourceFuncsConfig
}

func (c *KudoManagerCollector) Collect() error {

	kudoDiagRunner := &Runner{}
	p := &NonFailingPrinter{fs: c.fs}

	kudoDiagRunner.
		Run(&ResourceCollector{
			loadResourceFn: c.kr.Pods,
			errKind:        "pod",
			parentDir:      c.ctx.attachToRoot,
			callback:       c.ctx.mustAddPodNames,
			printer:        c.p,
			printMode:      ObjectListWithDirs}).
		Run(&ResourceCollector{
			loadResourceFn: c.kr.Services,
			errKind:        "service",
			parentDir:      c.ctx.attachToRoot,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.kr.StatefulSets,
			errKind:        "statefulset",
			parentDir:      c.ctx.attachToRoot,
			printer:        c.p,
			printMode:      RuntimeObject}).
		Run(&ResourceCollector{
			loadResourceFn: c.kr.ServiceAccounts,
			errKind:        "serviceaccount",
			parentDir:      c.ctx.attachToRoot,
			printer:        c.p,
			printMode:      RuntimeObject}).
		RunForEach(c.ctx.podNames, func(podName string) Collector {
			return &LogCollector{loadLogFn: c.kr.Log, podName: podName, parentDir: c.ctx.attachToRoot, printer: p}
		})
	return kudoDiagRunner.fatalErr
}
