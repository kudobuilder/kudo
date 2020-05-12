package diagnostics

import (
	"fmt"
	"io"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes/scheme"
)

const diagDir = "diag"

type printMode int

const (
	ObjectWithDir printMode = iota
	ObjectsWithDir
	RuntimeObject
)

type Printable interface {
	print(afero.Fs) error
}

type PrintableList []Printable

func (ps PrintableList) print(fs afero.Fs) error {
	for _, p := range ps {
		if err := p.print(fs); err != nil {
			return err // TODO:
		}
	}
	return nil
}

func NewPrintableObject(obj runtime.Object, baseDir func() string) (Printable, error) {
	o, ok := obj.(Object)
	if !ok {
		return nil, fmt.Errorf("kind %s doesn't have metadata", obj.GetObjectKind().GroupVersionKind().Kind)
	}
	ret := PrintableRuntimeObject{
		o:         o,
		parentDir: baseDir,
		ownDir:    func() string { return strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind) + "_" + o.GetName() },
		name:      func() string { return o.GetName() + ".yaml" },
	}
	return &ret, nil
}

func NewPrintableObjectList(obj runtime.Object, baseDir func() string) (Printable, error) {
	var ret PrintableList
	err := meta.EachListItem(obj, func(o runtime.Object) error {
		p, err := NewPrintableObject(o, baseDir)
		if err != nil {
			return err
		}
		ret = append(ret, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func NewPrintableRuntimeObject(obj runtime.Object, baseDir func() string) (Printable, error) {
	if meta.IsListType(obj) && meta.LenList(obj) == 0 {
		return nil, nil
	}
	ret := PrintableRuntimeObject{
		o:         obj,
		parentDir: baseDir,
		name:      func() string { return strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) + ".yaml" },
	}
	return &ret, nil
}

type PrintableLog struct {
	name      string
	log       io.ReadCloser
	parentDir func() string
}

func (p *PrintableLog) print(os afero.Fs) error {
	name := p.parentDir() + "/" + "pod_" + p.name + "/" + p.name + ".log.gz"
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	z := newGzipWriter(file, 2048)
	err = z.Write(p.log)
	if err != nil {
		return err
	}
	_ = p.log.Close()
	return nil
}

type PrintableRuntimeObject struct {
	o         runtime.Object
	parentDir func() string
	ownDir    func() string // path relative to parent
	name      func() string
}

func (p *PrintableRuntimeObject) print(fs afero.Fs) error {
	if !isKudoCR(p.o) {
		err := kudo.SetGVKFromScheme(p.o, scheme.Scheme)
		if err != nil {
			return err
		}
	}
	dir, name := p.parentDir(), p.name()
	if p.ownDir != nil {
		dir += "/" + p.ownDir()
		err := fs.MkdirAll(dir, 0700)
		if err != nil {
			return err
		}
	}
	file, err := fs.Create(dir + "/" + name)
	if err != nil {
		return err
	}
	printer := printers.YAMLPrinter{}
	return printer.PrintObj(p.o, file)
}

type AnyPrintable struct {
	name string
	dir  func() string
	v    interface{}
}

func (p *AnyPrintable) Collect() (Printable, error) {
	return p, nil
}

func (p *AnyPrintable) print(fs afero.Fs) error {
	file, err := fs.Create(p.dir() + "/" + p.name + ".yaml")
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(p.v)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}

type PrintableError struct {
	error
	Fatal bool
	name  string
	dir   func() string
}

func (p *PrintableError) print(fs afero.Fs) error {
	file, err := fs.Create(p.dir() + "/" + p.name + ".err")
	if err != nil {
		return err
	}
	b := []byte(p.Error())
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}
