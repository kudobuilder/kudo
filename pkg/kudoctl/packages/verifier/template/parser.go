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

/*
Package template manages parsing of Go templates used in KUDO for the purpose of evaluating fields used in the template files.
Deep knowledge of Go templates is necessary in order to fully understand how this package works.  Details for template and parsing available at
https://github.com/golang/go/tree/release-branch.go1.13/src/text/template and https://github.com/golang/go/tree/release-branch.go1.13/src/text/template/parse.

Core to the concept is the evaluation of different types of Nodes defined https://github.com/golang/go/blob/release-branch.go1.13/src/text/template/parse/node.go#L51-L71
The most significant node types that are important to KUDO and their descriptions are:

	parse.ActionNode
		This is a node such as {{ .Params.CUSTOM_CASSANDRA_YAML_BASE64 | b64dec }} where an evaluation is necessary.  The details needed KUDO are in the Pipeline (detailed below)
	parse.IfNode
		This a node such as {{ if .Params.CUSTOM_CASSANDRA_YAML_BASE64 }} which includes {{ end }}.  The node could have a body which needs to be processed in addition to the capturing of the subject of the if which is in the Pipeline.
	parse.WithNode
		Functions identically to IfNode however we don't have a working example.
	parse.ListNode
		Is a collection of nodes to process, an example is the Root node. When "walking" nodes "if" and "with" nodes could have lists of nodes.  This is a key node for traversal of the entire tree through recursion.
	parse.TextNode
		This is a node that is a body of text with no template fields / nodes to evaluate.

All other Node types are not supported at this time.
*/

// walkNodes walks the nodes of a template providing an array of parameters
// this function makes heavy use and deep understanding of Go Templating.   The details for
// different types of nodes and how they are parse is in `parse.Node`.
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
