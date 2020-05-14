package diagnostics

import (
	"fmt"
	"io"

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
	p              *NonFailingPrinter
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
		c.p.printError(err, c.parentDir(), c.errKind)
	case obj == nil || meta.IsListType(obj) && meta.LenList(obj) == 0:
		if c.failOnError {
			return fmt.Errorf("no object(s) of kind %s retrieved", c.errKind)
		}
	default:
		if c.callback != nil {
			c.callback(obj)
		}
		c.p.printObject(obj, c.parentDir(), c.printMode)
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
		if c.callback != nil {
			c.callback(obj)
		}
		objs[i] = obj
	}
	for i, c := range g {
		c.p.printObject(objs[i], c.parentDir(), modes[i])
	}
	return nil
}

// LogCollector - Collector interface implementation for logs for a pod
type LogCollector struct {
	loadLogFn func(string) (io.ReadCloser, error)
	podName   string
	parentDir func() string
	p         *NonFailingPrinter
}

// Collect - prints either a log or an error. Always returns nil error.
func (c *LogCollector) Collect() error {
	log, err := c.loadLogFn(c.podName)
	if err != nil {
		c.p.printError(err, c.parentDir(), fmt.Sprintf("%s.log", c.podName))
	} else {
		c.p.printLog(log, c.parentDir(), c.podName)
		_ = log.Close()
	}
	return nil
}
