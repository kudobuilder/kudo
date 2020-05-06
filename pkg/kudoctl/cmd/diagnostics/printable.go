package diagnostics

import (
	"io"
	"os"
)

type printable interface {
	GetName() string
	Print (w io.Writer) error
}

type printableTree struct {
	o        printable
	children []*printableTree
}

func printTree(t []*printableTree, dir string) error {
	return _print(t, "diag/" + dir)
}

func _print (nodes []*printableTree, dir string) error {
	if len(nodes) == 0 {
		return nil
	}
	for _, t := range nodes {
		kind := t.o.(Object).GetObjectKind().GroupVersionKind().Kind // TODO:
		nodeDir := dir + "/" + kind + "_" + t.o.GetName()
		err := os.MkdirAll(nodeDir, 0700)
		if err != nil {
			return err
		}
		name := t.o.GetName() + ".yml" //TODO:
		w, err := os.Create(nodeDir + "/" + name)
		if err !=nil {
			return err
		}
		err = t.o.Print(w)
		if err != nil {
		return err
	}
		err = _print(t.children, nodeDir)
		if err != nil {
			return err
		}
	}
	return nil
}