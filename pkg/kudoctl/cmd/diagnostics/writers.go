package diagnostics

import (
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

type objYamlWriter struct {
	obj runtime.Object
}

func (w objYamlWriter) write(file io.Writer) error {
	printer := printers.YAMLPrinter{}
	return printer.PrintObj(w.obj, file)
}

type byteWriter struct {
	b []byte
}

func (w byteWriter) write(file io.Writer) error {
	_, err := file.Write(w.b)
	return err
}

type gzipStreamWriter struct {
	stream io.ReadCloser
}

func (w gzipStreamWriter) write(file io.Writer) error {
	return newGzipWriter(file).write(w.stream)
}
