package template

import (
	"fmt"
	"strings"
	"text/template/parse"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// nodes are template nodes
type nodes struct {
	parameters     []string
	implicitParams []string
	error          *string
}

// nodeMap is a map of template files to template nodes
type nodeMap map[string]nodes

// getNodeMap converts a set of Templates to nodeMap which is a map of file names to template nodes
func getNodeMap(ts packages.Templates) nodeMap {

	tNodes := nodeMap{}

	for fname, file := range ts {

		e := renderer.New()
		t := e.Template(fname)
		// parse 1 template file using engine render template
		tplate, err := t.Parse(file)
		if err != nil {
			errMsg := fmt.Sprintf("template file %q reports the following error: %v", fname, err)
			n := nodes{
				error: &errMsg,
			}
			tNodes[fname] = n

			continue
		}

		//nodeMap is a map of node types ("Implicits", "Params") to a set of that type (which is go is a map :))
		nodeMap := map[string]map[string]bool{}
		walkNodes(tplate.Root, fname, nodeMap)

		n := nodes{
			parameters:     values(nodeMap, "Params"),
			implicitParams: values(nodeMap, "Implicits"),
		}
		tNodes[fname] = n
	}

	return tNodes
}

//walkNodes walks the nodes of a template providing an array of parameters
func walkNodes(node parse.Node, fname string, nodeMap map[string]map[string]bool) {
	switch node := node.(type) {
	case *parse.ActionNode:
		walkPipes(node.Pipe, nodeMap)
	//	if and with operate the same however we can't fail through in type switch
	case *parse.IfNode:
		walkNodes(node.List, fname, nodeMap)
		walkNodes(node.ElseList, fname, nodeMap)
		walkPipes(node.Pipe, nodeMap)
	case *parse.WithNode:
		walkNodes(node.List, fname, nodeMap)
		walkNodes(node.ElseList, fname, nodeMap)
		walkPipes(node.Pipe, nodeMap)
	case *parse.ListNode:
		if node == nil {
			return
		}
		for _, n := range node.Nodes {
			walkNodes(n, fname, nodeMap)
		}
	case *parse.RangeNode: // no support for Range, Template or TextNodes
	case *parse.TemplateNode:
	case *parse.TextNode:
	default:
		clog.V(2).Printf("file %q has unknown node: %s", fname, node)
	}
}

// walkPipes walks the pipes of specific block types which may contain params
func walkPipes(node *parse.PipeNode, nodeMap map[string]map[string]bool) {
	for _, cmd := range node.Cmds {
		for _, arg := range cmd.Args {
			switch n := arg.(type) {
			case *parse.FieldNode:
				// not evaluated
				if len(n.Ident) < 1 {
					return
				}
				//implicits have .Name which has 1 Indent
				if len(n.Ident) == 1 {
					addNodeSliceMap(nodeMap, "Implicits", trimNodeValue(arg.String()))
					return
				}
				//	others like .Params.Foo  are deeper.   We currently only support 1 deep.
				// .Params or similar is the key
				addNodeSliceMap(nodeMap, n.Ident[0], n.Ident[1])
				if len(n.Ident) > 2 {
					clog.V(3).Printf("template node %v has more elements than is supported", arg.String())
				}
			}
		}
	}
}

func ensureNodeMapFor(nodeMap map[string]map[string]bool, key string) {
	if _, ok := nodeMap[key]; !ok {
		nodeMap[key] = make(map[string]bool)
	}
}

func addNodeSliceMap(nodeMap map[string]map[string]bool, key string, value string) {
	ensureNodeMapFor(nodeMap, key)
	nodeMap[key][value] = true
}

func trimNodeValue(s string) string {
	return strings.TrimPrefix(s, ".")
}

// values takes a map of map node values and provides a slice of values
func values(nodeMap map[string]map[string]bool, key string) []string {
	var v []string
	for k := range nodeMap[key] {
		v = append(v, k)
	}
	return v
}
