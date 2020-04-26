package diagnostics

import (
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

type resourceHolder struct {
	obj     Object
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (h *resourceHolder) result() Object {
	return h.obj
}

type resourceListHolder struct {
	objs    []Object
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (h *resourceListHolder) result() []Object {
	return h.objs
}

type logHolder struct {
	logStream io.ReadCloser
	t infoType
	kind string
	metav1.Object
}

type describeHolder struct {
	desc string
	t infoType
	kind string
	metav1.Object
}