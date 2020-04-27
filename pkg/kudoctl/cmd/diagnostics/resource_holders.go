package diagnostics

import (
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

type pathHolder string

func (s *pathHolder) getPath() string {
	return string(*s)
}

func (s *pathHolder) setPath(str string) {
	*s = pathHolder(str)
}

type resourceHolder struct {
	pathHolder
	obj     *ObjectWithParent
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (h *resourceHolder) result() *ObjectWithParent {
	return h.obj
}

type resourceListHolder struct {
	pathHolder
	objs    []ObjectWithParent
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (h *resourceListHolder) result() []ObjectWithParent {
	return h.objs
}

type logHolder struct {
	pathHolder
	logStream io.ReadCloser
	t         infoType
	kind      string
	podParent *ObjectWithParent
	podName   string
	nameSpace string
}

type descriptionHolder struct {
	desc string
	t infoType
	kind string
	metav1.Object
}