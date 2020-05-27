package diagnostics

import (
	"io"

	"github.com/spf13/afero"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

type objYamlWriter struct {
	obj runtime.Object
}

func (w objYamlWriter) write(file afero.File) error {
	printer := printers.YAMLPrinter{}
	return printer.PrintObj(w.obj, file)
}

type byteWriter struct {
	b []byte
}

func (w byteWriter) write(file afero.File) error{
	_, err :=  file.Write(w.b)
	return err
}

type gzipStreamWriter struct {
	stream io.ReadCloser
}

func (w gzipStreamWriter) write(file afero.File) error{
	return newGzipWriter(file).write(w.stream)
}
