package diagnostics

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceCollector struct {
	resourceFn  func() (runtime.Object, error)
	errKind     string
	parentDir   func() string
	failOnError bool
	callback    func(o runtime.Object)
	p           *ObjectPrinter
	mode        printMode
}

func (c *ResourceCollector) Collect() error {
	obj, err := c.resourceFn()
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
		c.p.printObject(obj, c.parentDir(), c.mode)
	}
	return nil
}

type ResourceCollectorGroup []ResourceCollector

func (g ResourceCollectorGroup) Collect() error {
	objs := make([]runtime.Object, len(g))
	modes := make([]printMode, len(g))
	for i, c := range g {
		obj, err := c.resourceFn()
		if err != nil {
			return err // TODO: wrap
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

type LogCollector struct {
	r         *ResourceFuncsConfig
	podName   string
	parentDir func() string
	p         *ObjectPrinter
}

func (c *LogCollector) Collect() error {
	log, err := c.r.Log(c.podName)
	if err != nil {
		c.p.printError(err, c.parentDir(), fmt.Sprintf("%s.log", c.podName))
	} else {
		c.p.printLog(log, c.parentDir(), c.podName)
	}
	return nil
}
