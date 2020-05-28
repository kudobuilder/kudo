package diagnostics

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// Ensure collector is implemented
var (
	_ collector = &resourceCollector{}
	_ collector = &resourceCollectorGroup{}
	_ collector = &logsCollector{}
)

// resourceCollector - collector interface implementation for Kubernetes resources (runtime objects)
type resourceCollector struct {
	loadResourceFn func() (runtime.Object, error)
	name           string               // object kind used to describe the error
	parentDir      func() string        // parent dir to attach the printer's output
	failOnError    bool                 // define whether the collector should return the error
	callback       func(runtime.Object) // will be called with the retrieved resource after collection to update shared context
	printer        *nonFailingPrinter
	printMode      printMode
}

// collect - load a resource and send either the resource or collection error to printer
// return error if failOnError field is set to true
// if failOnError is true, finding no object(s) is treated as an error
func (c *resourceCollector) collect() error {
	obj, err := c._collect(c.failOnError)
	if err != nil {
		c.printer.printError(err, c.parentDir(), c.name)
		if c.failOnError {
			return err
		}
	}
	if obj != nil {
		c.printer.printObject(obj, c.parentDir(), c.printMode)
	}
	return nil
}

func (c *resourceCollector) _collect(failOnError bool) (runtime.Object, error) {
	obj, err := c.loadResourceFn()
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.name, err)
	case obj == nil || reflect.ValueOf(obj).IsNil() || (meta.IsListType(obj) && meta.LenList(obj) == 0):
		obj = nil
		if failOnError {
			return nil, fmt.Errorf("no object(s) of kind %s retrieved", c.name)
		}
	default:
		if c.callback != nil {
			c.callback(obj)
		}
	}
	return obj, nil
}

// resourceCollectorGroup - a composite collector for Kubernetes runtime objects whose loading and printing depend on
// each other's side-effects on the shared context
type resourceCollectorGroup struct {
	collectors []resourceCollector
	parentDir  func() string
}

// collect - collect resource and run callback for each collector, print all afterwards
// collection failures are treated as fatal regardless of the collectors failOnError flag setting
func (g resourceCollectorGroup) collect() error {
	objs := make([]runtime.Object, len(g.collectors))
	modes := make([]printMode, len(g.collectors))
	for i, c := range g.collectors {
		obj, err := c._collect(true)
		if err != nil {
			c.printer.printError(err, g.parentDir(), c.name)
			return err
		}
		objs[i], modes[i] = obj, c.printMode
	}
	for i, c := range g.collectors {
		c.printer.printObject(objs[i], c.parentDir(), modes[i])
	}
	return nil
}

type logsCollector struct {
	loadLogFn func(string, string) (io.ReadCloser, error)
	pods      []v1.Pod
	parentDir func() string
	printer   *nonFailingPrinter
}

func (c *logsCollector) collect() error {
	for _, pod := range c.pods {
		for _, container := range pod.Spec.Containers {
			log, err := c.loadLogFn(pod.Name, container.Name)
			if err != nil {
				c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), fmt.Sprintf("%s.log", container.Name))
			} else {
				c.printer.printLog(log, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), container.Name)
				_ = log.Close()
			}
		}
	}
	return nil
}
