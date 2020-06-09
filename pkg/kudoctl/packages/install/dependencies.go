package install

import (
	"fmt"

	"github.com/thoas/go-funk"
	"github.com/yourbasic/graph"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
)

// dependencyGraph is modeled after 'graph.Mutable' but allows to add vertices.
type dependencyGraph struct {
	edges []map[int]struct{}
}

// AddVertex adds a new vertex to the dependency graph.
func (g *dependencyGraph) AddVertex() {
	g.edges = append(g.edges, map[int]struct{}{})
}

// AddEdge adds an edge from vertex v to w to the dependency graph.
func (g *dependencyGraph) AddEdge(v, w int) {
	g.edges[v][w] = struct{}{}
}

// Order returns the number of vertices of the dependency graph.
func (g *dependencyGraph) Order() int {
	return len(g.edges)
}

func (g *dependencyGraph) Visit(v int, do func(w int, c int64) bool) bool {
	for w := range g.edges[v] {
		if do(w, 1) {
			return true
		}
	}

	return false
}

// gatherDependencies resolved all dependencies of a package.
// Dependencies are resolved recursively.
// Cyclic dependencies are detected and result in an error.
func gatherDependencies(root packages.Resources, resolver pkgresolver.Resolver) ([]packages.Resources, error) {
	pkgs := []packages.Resources{
		root,
	}

	// Each vertex in 'g' matches an index in 'pkgs'.
	g := dependencyGraph{
		edges: []map[int]struct{}{{}},
	}

	if err := dependencyWalk(&pkgs, &g, root, resolver); err != nil {
		return nil, err
	}

	// Remove 'root' from the list of dependencies.
	pkgs = funk.Drop(pkgs, 1).([]packages.Resources) //nolint:errcheck

	return pkgs, nil
}

func dependencyWalk(
	pkgs *[]packages.Resources,
	g *dependencyGraph,
	parent packages.Resources,
	resolver pkgresolver.Resolver) error {
	//nolint:errcheck
	operatorTasks := funk.Filter(parent.OperatorVersion.Spec.Tasks, func(task v1beta1.Task) bool {
		return task.Kind == engtask.KudoOperatorTaskKind
	}).([]v1beta1.Task)

	versionOf := func(pkg packages.Resources) func(packages.Resources) bool {
		return func(r packages.Resources) bool {
			return r.Operator.Name == pkg.Operator.Name &&
				r.OperatorVersion.Spec.AppVersion == pkg.OperatorVersion.Spec.AppVersion &&
				r.OperatorVersion.Spec.Version == pkg.OperatorVersion.Spec.Version
		}
	}

	for _, operatorTask := range operatorTasks {
		dependency, err := resolver.Resolve(
			operatorTask.Spec.Package,
			operatorTask.Spec.AppVersion,
			operatorTask.Spec.OperatorVersion)
		if err != nil {
			return fmt.Errorf(
				"failed to resolve package %s-%s-%s, dependency of package %s-%s-%s: %v",
				operatorTask.Spec.Package,
				operatorTask.Spec.AppVersion,
				operatorTask.Spec.OperatorVersion,
				parent.Operator.Name,
				parent.OperatorVersion.Spec.AppVersion,
				parent.OperatorVersion.Spec.Version,
				err)
		}

		parentIndex := funk.IndexOf(*pkgs, funk.Find(*pkgs, versionOf(parent)))
		if parentIndex == -1 {
			panic("failed to find parent index in dependency graph")
		}

		if funk.Find(*pkgs, versionOf(*dependency.Resources)) == nil {
			clog.Printf(
				"Adding new dependency %s-%s-%s",
				dependency.Resources.Operator.Name,
				dependency.Resources.OperatorVersion.Spec.AppVersion,
				dependency.Resources.OperatorVersion.Spec.Version)

			*pkgs = append(*pkgs, *dependency.Resources)

			// The number of vertices in 'g' has to match the number of packages
			// we're tracking.
			g.AddVertex()

			if err := dependencyWalk(pkgs, g, *dependency.Resources, resolver); err != nil {
				return err
			}
		}

		pkgIndex := funk.IndexOf(*pkgs, funk.Find(*pkgs, versionOf(*dependency.Resources)))
		if pkgIndex == -1 {
			panic("failed to find package index in dependency graph")
		}

		// This is a directed graph. The edge represents a dependency of
		// the parent package on the current package.
		g.AddEdge(parentIndex, pkgIndex)

		if !graph.Acyclic(g) {
			return fmt.Errorf(
				"cyclic package dependency found when adding package %s-%s-%s",
				dependency.Resources.Operator.Name,
				dependency.Resources.OperatorVersion.Spec.AppVersion,
				dependency.Resources.OperatorVersion.Spec.Version)
		}
	}

	return nil
}
