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
			return err
		}
	}
	return nil
}

// NewPrintableObject - create a wrapper to print runtime.Object in its own directory based on its metadata
// fails if the object does not implement metav1.Object
func NewPrintableObject(obj runtime.Object, parentDir func() string) (Printable, error) {
	o, ok := obj.(Object)
	if !ok {
		return nil, fmt.Errorf("kind %s doesn't have metadata", obj.GetObjectKind().GroupVersionKind().Kind)
	}
	ret := PrintableRuntimeObject{
		o:              o,
		parentDir:      parentDir,
		relToParentDir: func() string { return strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind) + "_" + o.GetName() },
		name:           func() string { return o.GetName() + ".yaml" },
	}
	return &ret, nil
}

// NewPrintableObjectList - wrappers of runtime.Object, so that each should be printed in its own directory based on its metadata
// fails if the object is not a list or if any of the items does not implement metav1.Object
func NewPrintableObjectList(obj runtime.Object, parentDir func() string) (Printable, error) {
	var ret PrintableList
	err := meta.EachListItem(obj, func(o runtime.Object) error {
		p, err := NewPrintableObject(o, parentDir)
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

// NewPrintableRuntimeObject - wrapper to print runtime.Object as a file in the parent's directory
func NewPrintableRuntimeObject(obj runtime.Object, parentDir func() string) (Printable, error) {
	if meta.IsListType(obj) && meta.LenList(obj) == 0 {
		return nil, nil
	}
	ret := PrintableRuntimeObject{
		o:         obj,
		parentDir: parentDir,
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

// PrintableRuntimeObject - printable implementation for runtime.Object
// name and directory information is packed into lambdas so that the object could be created before this data becomes available
type PrintableRuntimeObject struct {
	o              runtime.Object
	parentDir      func() string
	relToParentDir func() string
	name           func() string
}

func (p *PrintableRuntimeObject) print(fs afero.Fs) error {
	if !isKudoCR(p.o) {
		err := kudo.SetGVKFromScheme(p.o, scheme.Scheme)
		if err != nil {
			return err
		}
	}
	dir, name := p.parentDir(), p.name()
	if p.relToParentDir != nil {
		dir += "/" + p.relToParentDir()
		err := fs.MkdirAll(dir, 0700)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	fileWithPath := fmt.Sprintf("%s/%s", dir, name)
	file, err := fs.Create(fileWithPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %v", fileWithPath, err)
	}
	printer := printers.YAMLPrinter{}
	return printer.PrintObj(p.o, file)
}

// PrintableYaml - printable implementation to print anything as yaml
// implements collector for convenience
type PrintableYaml struct {
	name string
	dir  func() string
	v    interface{}
}

func (p *PrintableYaml) Collect() (Printable, error) {
	return p, nil
}

func (p *PrintableYaml) print(fs afero.Fs) error {
	b, err := yaml.Marshal(p.v)
	if err != nil {
		return fmt.Errorf("failed to marshal object to %s/%s.yaml: %v", p.dir(), p.name, err)
	}
	fileNameWithPath := fmt.Sprintf("%s/%s.yaml", p.dir(), p.name)
	return printBytes(fs, b, fileNameWithPath)
}

type PrintableError struct {
	error
	Fatal bool
	name  string
	dir   func() string
}

func (p *PrintableError) print(fs afero.Fs) error {
	b := []byte(p.Error())
	return printBytes(fs, b, p.dir()+"/"+p.name+".err")

}

func printBytes(fs afero.Fs, b []byte, fileName string) error {
	file, err := fs.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", fileName, err)
	}
	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %v", fileName, err)
	}
	return nil
}
