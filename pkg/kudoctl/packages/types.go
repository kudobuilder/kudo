package packages

import (
	"fmt"
	"strings"
	"text/template/parse"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

const (
	APIVersion = "kudo.dev/v1beta1"
)

var (
	// Implicits is a set of usable implicits
	// defined in render.go
	Implicits = map[string]bool{
		"Name":         true, // instance name
		"Namespace":    true,
		"OperatorName": true,
		"Params":       true,
		"PlanName":     true,
		"PhaseName":    true,
		"StepName":     true,
		"AppVersion":   true,
	}
)

// This is an abstraction which abstracts the underlying packages, which is likely file system or compressed file.
// There should be a complete separation between retrieving a packages if not local and working with a packages.

// Package is an abstraction of the collection of files that makes up a package.  It is anything we can retrieve the Resources from.
type Package struct {
	// transformed server view
	Resources *Resources
	// working with local package files
	Files *Files
}

// Resources is collection of CRDs that are used when installing operator
// during installation, package format is converted to this structure
type Resources struct {
	Operator        *v1beta1.Operator
	OperatorVersion *v1beta1.OperatorVersion
	Instance        *v1beta1.Instance
}

type Parameter []v1beta1.Parameter

// Len returns the number of params.
// This is needed to allow sorting of params.
func (p Parameter) Len() int { return len(p) }

// Swap swaps the position of two items in the params slice.
// This is needed to allow sorting of params.
func (p Parameter) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Less returns true if the name of a param a is less than the name of param b.
// This is needed to allow sorting of params.
func (p Parameter) Less(x, y int) bool {
	return p[x].Name < p[y].Name
}

// Templates is a map of file names and stringified files in the template folder of an operator
type Templates map[string]string

// Nodes are template nodes
type Nodes struct {
	Parameters     []string
	ImplicitParams []string
	Error          *string
}

// TemplateNodes is a map of template files to template nodes
type TemplateNodes map[string]Nodes

// Nodes converts a set of Templates to TemplateNodes which is a map of file names to template nodes
func (ts Templates) Nodes() TemplateNodes {

	tNodes := TemplateNodes{}

	for fname, file := range ts {
		// template nodes to be collected in a set
		// fresh for each file

		e := renderer.New()
		t := e.Template(fname)
		// parse 1 template
		tplate, err := t.Parse(file)
		if err != nil {
			errMsg := fmt.Sprintf("template file %q reports the following error: %v", fname, err)
			n := Nodes{
				Error: &errMsg,
			}
			tNodes[fname] = n

			continue
		}
		// cycle through all the nodes and collect Action nodes
		//nodeMap is a map of node types ("implicit", "Params") to a set of that type (which is go is a map :))
		nodeMap := map[string]map[string]bool{}
		walkNodes(tplate.Root, fname, nodeMap)

		n := Nodes{
			Parameters:     values(nodeMap, "Params"),
			ImplicitParams: values(nodeMap, "Implicits"),
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

// Files represents the raw operator package format the way it is found in the tgz packages
type Files struct {
	Templates Templates
	Operator  *OperatorFile
	Params    *ParamsFile
}

// ParamsFile is a representation of the package params.yaml
type ParamsFile struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Parameters Parameter `json:"parameters"`
}

// OperatorFile is a representation of the package operator.yaml
type OperatorFile struct {
	APIVersion        string                  `json:"apiVersion,omitempty"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description,omitempty"`
	Version           string                  `json:"version"`
	AppVersion        string                  `json:"appVersion,omitempty"`
	KUDOVersion       string                  `json:"kudoVersion,omitempty"`
	KubernetesVersion string                  `json:"kubernetesVersion,omitempty"`
	Maintainers       []*v1beta1.Maintainer   `json:"maintainers,omitempty"`
	URL               string                  `json:"url,omitempty"`
	Tasks             []v1beta1.Task          `json:"tasks"`
	Plans             map[string]v1beta1.Plan `json:"plans"`
}
