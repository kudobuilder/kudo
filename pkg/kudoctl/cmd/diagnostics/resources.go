package diagnostics

/* collectors for kubernetes resources and related data, e.g logs and "describes" */

import (
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
	"k8s.io/kubectl/pkg/describe/versioned"
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
}

type objectLister func() ([]runtime.Object, error)

type objectGetter func() (runtime.Object, error)

type resourceHolder struct {
	obj     runtime.Object
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (h *resourceHolder) result() runtime.Object {
	return h.obj
}

func (h *resourceHolder) print(f writerFactory) error {
	meta := h.obj.(v1.ObjectMetaAccessor).GetObjectMeta() // TODO: handle error?
	info := resourceInfo{
		T:         ResourceInfoType,
		Kind:      h.obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
	}
	w, err := f(info)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = h.printer.PrintObj(h.obj, w)
	if err != nil {
		return err
	}
	return nil
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

type resourceListHolder struct {
	objs    []runtime.Object
	printer printers.YAMLPrinter // TODO: allow other printers
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

func (h *resourceListHolder) print(f writerFactory) error {
	for _, obj := range h.objs {
		meta := obj.(v1.ObjectMetaAccessor).GetObjectMeta() // TODO: handle error?
		info := resourceInfo{
			T:         ResourceInfoType,
			Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
			Namespace: meta.GetNamespace(),
			Name:      meta.GetName(),
		}
		w, err := f(info)
		if err != nil {
			fmt.Println(err)
			return err
		}
		err = h.printer.PrintObj(obj, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *resourceListHolder) result() []runtime.Object {
	return h.objs
}

type describeListCollector struct {
	config *restclient.Config
	d      *resourceListCollector
}

func (c *describeListCollector) Collect(f writerFactory) error {
	for _, item := range c.d.result() {
		describer, _ := versioned.DescriberFor(item.GetObjectKind().GroupVersionKind().GroupKind(), c.config)
		meta := item.(v1.ObjectMetaAccessor).GetObjectMeta()
		desc, _ := describer.Describe(meta.GetNamespace(), meta.GetName(), describe.DescriberSettings{ShowEvents: true})

		info := resourceInfo{
			T:         DescribeInfoType,
			Kind:      item.GetObjectKind().GroupVersionKind().Kind,
			Namespace: meta.GetNamespace(),
			Name:      meta.GetName(),
		}
		w, err := f(info)
		if err != nil {
			fmt.Println(err)
			// TODO: handle error, collect errors
			return err
		}
		_, err = fmt.Fprintf(w, desc)
		if err != nil {
			return err // TODO: collect errors
		}
	}
	return nil
}

type describeCollector struct {
	config *restclient.Config
	d      *resourceCollector
}

func (c *describeCollector) Collect(f writerFactory) error {
	item := c.d.result()
	describer, ok := versioned.DescriberFor(item.GetObjectKind().GroupVersionKind().GroupKind(), c.config)
	if !ok {
		return nil // TODO: describing only standard types may not be quite OK
	}
	meta := item.(v1.ObjectMetaAccessor).GetObjectMeta()
	desc, _ := describer.Describe(meta.GetNamespace(), meta.GetName(), describe.DescriberSettings{ShowEvents: true})

	info := resourceInfo{
		T:         DescribeInfoType,
		Kind:      item.GetObjectKind().GroupVersionKind().Kind,
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
	}
	w, err := f(info)
	if err != nil {
		fmt.Println(err)
		// TODO: handle error, collect errors
		return err
	}
	_, err = fmt.Fprintf(w, desc)
	return err
}

type logCollector struct {
	*kube.Client
	ns   string
	opts corev1.PodLogOptions
	logs map[string]io.ReadCloser // TODO: GVK as a key
	pods *resourceListCollector
}

func (c *logCollector) Collect(f writerFactory) error {
	var err error
	for _, p := range c.pods.result() {
		pod := p.(*corev1.Pod)
		c.logs[pod.Name], err = c.KubeClient.CoreV1().Pods(c.ns).GetLogs(pod.Name, &c.opts).Stream()
		if err != nil {
			return err // TODO: collect errors
		}
	}
	if err == nil {
		err = c.print(f)
	}
	return err
}

func (c *logCollector) print(f writerFactory) error {
	for name, log := range c.logs {
		info := resourceInfo{
			T:         LogInfoType,
			Namespace: c.ns,
			Kind:      "pod",
			Name:      name,
		}
		w, err := f(info)
		if err != nil {
			fmt.Println(err)
			return err
		}
		z := newGzipWriter(w, 2048)
		z.Write(log) //TODO: get error, return error
		log.Close()
	}
	return nil
}
