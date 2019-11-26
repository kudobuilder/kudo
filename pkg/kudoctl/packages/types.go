package packages

import (
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
		nodeSet := map[string]bool{}
		t := template.New(fname)
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
		nodes := walkNodes(tplate.Root, fname)
		for _, node := range nodes {
			tnode := trimNode(node)
			nodeSet[tnode] = true
		}

		n := Nodes{
			Parameters:     parameters(nodeSet),
			ImplicitParams: implicits(nodeSet),
		}
		tNodes[fname] = n
	}

	return tNodes
}

//walkNodes walks the nodes of a template providing an array of parameters
func walkNodes(node parse.Node, fname string) []string {
	switch node := node.(type) {
	case *parse.ActionNode:
		return []string{trimNode(node.String())}
	//	if and with operate the same however we can't fail through in type switch
	case *parse.IfNode:
		list := walkNodes(node.List, fname)
		elist := walkNodes(node.ElseList, fname)
		list = append(list, elist...)
		clist := walkPipes(node.Pipe)
		return append(list, clist...)
	case *parse.WithNode:
		list := walkNodes(node.List, fname)
		elist := walkNodes(node.ElseList, fname)
		list = append(list, elist...)
		clist := walkPipes(node.Pipe)
		return append(list, clist...)
	case *parse.ListNode:
		list := []string{}
		if node == nil {
			return list
		}
		for _, n := range node.Nodes {
			list = append(list, walkNodes(n, fname)...)
		}
		return list
	case *parse.RangeNode: // no support for Range, Template or TextNodes
	case *parse.TemplateNode:
	case *parse.TextNode:
	default:
		clog.V(2).Printf("file %q has unknown node: %s", fname, node)
	}
	return []string{}
}

// walkPipes walks the pipes of specific block types which may contain params
func walkPipes(node *parse.PipeNode) []string {
	params := []string{}
	for _, cmd := range node.Cmds {
		for _, arg := range cmd.Args {
			switch arg.(type) {
			case *parse.FieldNode:
				if strings.HasPrefix(arg.String(), ".Params") {
					params = append(params, arg.String())
				}
			}
		}
	}
	return params
}

// parameters takes a map and returns Nodes (which is a []string).
// map is used for set functionality (a goism)
// parameters converts array of Nodes to just nodes of parameters (those prefixed with "Params.") and strips the prefix
func parameters(nodes map[string]bool) []string {

	fields := []string{}

	for k := range nodes {
		if strings.Contains(k, "Params.") {
			fields = append(fields, trimParam(k))
		}
	}
	return fields
}

// implicits takes a map and returns implicits (which is a []string).
// map is used for set functionality (a goism)
func implicits(nodes map[string]bool) []string {

	fields := []string{}

	for k := range nodes {
		if !strings.Contains(k, ".Params.") {
			fields = append(fields, strings.TrimPrefix(k, "."))
		}
	}
	return fields
}

// takes string with "{{." prefix and "}}" suffix and removes prefix and suffix
func trimNode(s string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, "{{"), "}}")
}

// takes string with "Params." prefix and removes prefix
func trimParam(s string) string {
	return strings.TrimPrefix(s, ".Params.")
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
