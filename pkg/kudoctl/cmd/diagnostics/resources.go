package diagnostics

/* collectors for kubernetes resources and related data, e.g logs and "describes" */

import (
	"fmt"
)

type infoType int

const (
	ResourceInfoType = iota
	LogInfoType
	DescribeInfoType
)

type resourceInfo struct {
	T         infoType
	Namespace string
	Kind      string // TODO: should be GVK, not just Kind
	Name      string
	Parent *ObjectWithParent
}

type resourceCollector struct {
	getResource objectGetter
	resourceHolder
}

func (c *resourceCollector) Collect(f writerFactory) error {
	var err error
	if c.obj, err = c.getResource(); err == nil {
		err = c.print(f)
	}
	return err
}

func (c *resourceCollector) print(f writerFactory) error {
	meta := c.obj
	info := resourceInfo{
		T:         ResourceInfoType,
		Kind:      c.obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
		Parent: c.obj.parent,
	}
	w, err := f(info)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = c.printer.PrintObj(c.obj, w)
	if err != nil {
		return err
	}
	return nil
}

type resourceListCollector struct {
	getResources objectLister
	resourceListHolder
}

func (c *resourceListCollector) Collect(f writerFactory) error {
	var err error
	if c.objs, err = c.getResources(); err == nil {
		err = c.print(f)
	}
	return err
}

func (c *resourceListCollector) print(f writerFactory) error {
	for _, obj := range c.objs {
		info := resourceInfo{
			T:         ResourceInfoType,
			Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
			Parent: obj.parent,
		}
		w, err := f(info)
		if err != nil {
			fmt.Println(err)
			return err // TODO: collect errors
		}
		err = c.printer.PrintObj(obj, w)
		if err != nil {
			return err
		}
	}
	return nil
}

type describeListCollector struct {
	objs []descriptionHolder
	getDescribes      func() ([]descriptionHolder, error)
}

func (c *describeListCollector) Collect(writerFor writerFactory) error {
	var err error
	if c.objs, err = c.getDescribes(); err == nil {
		err = c.print(writerFor)
	}
	return err
}

func (c *describeListCollector) print (writerFor writerFactory) error {
	for _, obj := range c.objs {
		info := resourceInfo{
			T:         DescribeInfoType,
			Namespace: obj.GetNamespace(),
			Kind:      obj.kind,
			Name:      obj.GetName(),
		}
		w, err := writerFor(info)
		if err != nil {
			fmt.Println(err)
			// TODO: handle error, collect errors
			return err
		}
		_, err = fmt.Fprintf(w, obj.desc)
		if err != nil {
			return err // TODO: collect errors
		}
	}
	return nil
}

type describeCollector struct {
	obj *descriptionHolder
	getDescribe      func() (*descriptionHolder, error)
}

func (c *describeCollector) Collect(wf writerFactory) error {
	var err error
	if c.obj, err = c.getDescribe(); err == nil {
		err = c.print(wf)
	}
	return err
}

func (c *describeCollector) print (writerFor writerFactory) error {
		info := resourceInfo{
			T:         DescribeInfoType,
			Namespace: c.obj.GetNamespace(),
			Kind:      c.obj.kind,
			Name:      c.obj.GetName(),
		}
		w, err := writerFor(info)
		if err != nil {
			fmt.Println(err)
			return err
		}
		_, err = fmt.Fprintf(w, c.obj.desc)
		if err != nil {
			return err
		}
	return nil
}

type logCollector struct {
	logs []logHolder
	getLogs func() ([]logHolder, error)
}

func (c *logCollector) Collect(f writerFactory) error {
	var err error
	c.logs, err = c.getLogs()
	if err == nil {
		err = c.print(f)
	}
	return err
}

func (c *logCollector) print(f writerFactory) error {
	for _, log := range c.logs {
		info := resourceInfo{
			T:         LogInfoType,
			Namespace: log.nameSpace,
			Kind:      "pod",
			Name:      log.podName,
			Parent: log.podParent,
		}
		w, err := f(info)
		if err != nil {
			fmt.Println(err)
			return err
		}
		z := newGzipWriter(w, 2048)
		_ = z.Write(log.logStream) //TODO: get error, return error
		_ = log.logStream.Close()
	}
	return nil
}
