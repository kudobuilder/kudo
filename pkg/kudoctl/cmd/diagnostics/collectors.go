package diagnostics

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

// Ensure collector is implemented
var _ collector = &resourceCollector{}

// resourceCollector - collector interface implementation for Kubernetes resources (runtime objects)
type resourceCollector struct {
	loadResourceFn func() (runtime.Object, error)
	name           string               // object kind used to describe the error
	parentDir      stringGetter         // parent dir to attach the printer's output
	failOnError    bool                 // define whether the collector should return the error
	callback       func(runtime.Object) // will be called with the retrieved resource after collection to update shared context
	printMode      printMode
}

// collect - load a resource and send either the resource or collection error to printer
// return error if failOnError field is set to true
// if failOnError is true, finding no object(s) is treated as an error
func (c *resourceCollector) collect(printer *nonFailingPrinter) error {
	clog.V(4).Printf("Collect Resource %s in parent dir %s", c.name, c.parentDir())
	obj, err := c._collect(c.failOnError)
	if err != nil {
		printer.printError(err, c.parentDir(), c.name)
		if c.failOnError {
			return err
		}
	}
	if obj != nil {
		printer.printObject(obj, c.parentDir(), c.printMode)
	}
	return nil
}

func emptyResult(obj runtime.Object) bool {
	return obj == nil || reflect.ValueOf(obj).IsNil() || (meta.IsListType(obj) && meta.LenList(obj) == 0)
}

func (c *resourceCollector) _collect(failOnError bool) (runtime.Object, error) {
	obj, err := c.loadResourceFn()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.name, err)
	}
	if emptyResult(obj) {
		if failOnError {
			return nil, fmt.Errorf("no object(s) of kind %s retrieved", c.name)
		}
		return nil, nil
	}
	if c.callback != nil {
		c.callback(obj)
	}
	return obj, nil
}

// Ensure collector is implemented
var _ collector = &resourceCollectorGroup{}

// resourceCollectorGroup - a composite collector for Kubernetes runtime objects whose loading and printing depend on
// each other's side-effects on the shared context
type resourceCollectorGroup struct {
	collectors []resourceCollector
}

// collect - collect resource and run callback for each collector, print all afterwards
// collection failures are treated as fatal regardless of the collectors failOnError flag setting
func (g resourceCollectorGroup) collect(printer *nonFailingPrinter) error {
	clog.V(0).Printf("Collect ResourceGroup for %d collectors", len(g.collectors))
	objs := make([]runtime.Object, len(g.collectors))
	for i, c := range g.collectors {
		obj, err := c._collect(true)
		if err != nil {
			printer.printError(err, c.parentDir(), c.name)
			return err
		}
		objs[i] = obj
	}
	for i, c := range g.collectors {
		printer.printObject(objs[i], c.parentDir(), c.printMode)
	}
	return nil
}

// Ensure collector is implemented
var _ collector = &logsCollector{}

type logsCollector struct {
	loadLogFn func(string, string) (io.ReadCloser, error)
	pods      func() []v1.Pod
	parentDir stringGetter
}

func (c *logsCollector) collect(printer *nonFailingPrinter) error {
	clog.V(0).Printf("Collect Logs for %d pods", len(c.pods()))
	for _, pod := range c.pods() {
		for _, container := range pod.Spec.Containers {
			log, err := c.loadLogFn(pod.Name, container.Name)
			if err != nil {
				printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), fmt.Sprintf("%s.log", container.Name))
			} else {
				printer.printLog(log, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), container.Name)
				_ = log.Close()
			}
		}
	}
	return nil
}

var _ collector = &objCollector{}

type objCollector struct {
	obj       interface{}
	parentDir stringGetter
	name      string
}

func (c *objCollector) collect(printer *nonFailingPrinter) error {
	printer.printYaml(c.obj, c.parentDir(), c.name)
	return nil
}
