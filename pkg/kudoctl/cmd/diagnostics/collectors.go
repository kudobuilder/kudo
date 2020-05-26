package diagnostics

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// resourceCollector - collector interface implementation for Kubernetes resources (runtime objects)
type resourceCollector struct {
	loadResourceFn func() (runtime.Object, error)
	name           string               // object kind used to describe the error
	parentDir      func() string        // parent dir to attach the printer's output
	failOnError    bool                 // define whether the collector should return the error
	callback       func(runtime.Object) // will be called with the retrieved resource after cllection to update  shared context
	printer        *nonFailingPrinter
	printMode      printMode
}

// collect - load a resource and send either the resource or collection error to printer
// return error if failOnError field is set to true
// if failOnError is true, finding no object(s) is treated as an error
func (c *resourceCollector) collect() error {
	obj, err := c.loadResourceFn()
	switch {
	case err != nil:
		if c.failOnError {
			return fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.name, err)
		}
		c.printer.printError(err, c.parentDir(), c.name)
	case obj == nil || reflect.ValueOf(obj).IsNil() || meta.IsListType(obj) && meta.LenList(obj) == 0:
		if c.failOnError {
			return fmt.Errorf("no object(s) of kind %s retrieved", c.name)
		}
	default:
		if c.callback != nil {
			c.callback(obj)
		}
		c.printer.printObject(obj, c.parentDir(), c.printMode)
	}
	return nil
}

// resourceCollectorGroup - a composite collector for Kubernetes runtime objects whose loading and printing depend on
// each other's side-effects on the shared context
type resourceCollectorGroup []resourceCollector

// collect - collect resource and run callback for each collector, print all afterwards
// collection failures are treated as fatal regardless of the collectors failOnError flag setting
func (g resourceCollectorGroup) collect() error {
	objs := make([]runtime.Object, len(g))
	modes := make([]printMode, len(g))
	for i, c := range g {
		obj, err := c.loadResourceFn()
		if err != nil {
			return fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.name, err)
		}
		if obj == nil || reflect.ValueOf(obj).IsNil() || meta.IsListType(obj) && meta.LenList(obj) == 0 {
			return fmt.Errorf("no object(s) of kind %s retrieved", c.name)
		}
		if c.callback != nil {
			c.callback(obj)
		}
		objs[i], modes[i] = obj, c.printMode
	}
	for i, c := range g {
		c.printer.printObject(objs[i], c.parentDir(), modes[i])
	}
	return nil
}

// logCollector - collector interface implementation for logs for a pod
type logCollector struct {
	loadLogFn func(string) (io.ReadCloser, error)
	podName   string
	parentDir func() string
	printer   *nonFailingPrinter
}

// collect - prints either a log or an error. Always returns nil error.
func (c *logCollector) collect() error {
	log, err := c.loadLogFn(c.podName)
	if err != nil {
		c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", c.podName)), fmt.Sprintf("%s.log", c.podName))
	} else {
		c.printer.printLog(log, c.parentDir(), c.podName)
		_ = log.Close()
	}
	return nil
}
