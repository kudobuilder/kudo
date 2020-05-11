package diagnostics

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/afero"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
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
	return &PrintableObject{o: o, parentDir: baseDir}, nil
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
	return &PrintableRuntimeObject{o: obj, parentDir: baseDir}, nil
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
	_ = z.Write(p.log) //TODO: get error, return error
	_ = p.log.Close()
	return nil
}

type PrintableObject struct {
	o         Object
	parentDir func() string
	//printer printers.YAMLPrinter
}

func (p *PrintableObject) print(fs afero.Fs) error {
	return printObject(fs, p.o, p.parentDir(), &printers.YAMLPrinter{})
}

type PrintableRuntimeObject struct {
	o         runtime.Object
	parentDir func() string
	//printer printers.YAMLPrinter
}

func (p *PrintableRuntimeObject) print(fs afero.Fs) error {
	return printRuntimeObject(fs, p.o, p.parentDir(), &printers.YAMLPrinter{})
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
	return printAnyAsYaml(fs, p.v, p.dir(), p.name)
}

func printObject(os afero.Fs, o Object, dir string, printer printers.ResourcePrinter) error {
	if !isKudoCR(o) {
		err := SetGVKFromScheme(o)
		if err != nil {
			return err
		}
	}
	pDir := dir + "/" + strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind) + "_" + o.GetName()
	name := pDir + "/" + o.GetName() + ".yaml"
	err := os.MkdirAll(pDir, 0700)
	if err != nil {
		return err
	}
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	err = printer.PrintObj(o, file)
	if err != nil {
		return err
	}
	return nil
}

func printRuntimeObject(os afero.Fs, o runtime.Object, dir string, printer printers.ResourcePrinter) error {
	err := SetGVKFromScheme(o)
	if err != nil {
		return err
	}
	name := dir + "/" + strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind) + ".yaml"
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	err = printer.PrintObj(o, file)
	if err != nil {
		return err
	}
	return nil
}

func printAnyAsYaml(os afero.Fs, o interface{}, dir, name string) error {
	file, err := os.Create(dir + "/" + name + ".yaml")
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}
