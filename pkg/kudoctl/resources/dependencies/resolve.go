package dependencies

import (
	"fmt"

	"github.com/thoas/go-funk"
	"github.com/yourbasic/graph"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
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

type Dependency struct {
	packages.Resources
}

// Resolve resolves all dependencies of an OperatorVersion. Dependencies are resolved recursively.
// Cyclic dependencies are detected and result in an error. operatorArgument parameter is string that
// the user passed to install/upgrade an operator which is used to determine whether this is a local
// operator in a directory. In that case all its relative dependencies (if any exist) will be relative
// to the operator directory (the one with `operator.yaml` file).
// See github.com/kudobuilder/kudo/issues/1701 for additional context.
func Resolve(operatorVersion *kudoapi.OperatorVersion, resolver packages.Resolver) ([]Dependency, error) {
	root := packages.Resources{
		OperatorVersion: operatorVersion,
	}

	dependencies := []Dependency{
		{Resources: root},
	}

	// Each vertex in 'g' matches an index in 'dependencies'.
	g := dependencyGraph{
		edges: []map[int]struct{}{{}},
	}

	if err := dependencyWalk(&dependencies, &g, &root, 0, resolver); err != nil {
		return nil, err
	}

	// Remove 'root' from the list of dependencies.
	return dependencies[1:], nil
}

func dependencyWalk(
	dependencies *[]Dependency,
	g *dependencyGraph,
	parent *packages.Resources,
	parentIndex int,
	resolver packages.Resolver) error {
	//nolint:errcheck
	childrenTasks := funk.Filter(parent.OperatorVersion.Spec.Tasks, func(task kudoapi.Task) bool {
		return task.Kind == engtask.KudoOperatorTaskKind
	}).([]kudoapi.Task)

	for _, childTask := range childrenTasks {
		childPackageName := childTask.Spec.KudoOperatorTaskSpec.Package

		childResolved, err := resolver.Resolve(
			childPackageName,
			childTask.Spec.KudoOperatorTaskSpec.AppVersion,
			childTask.Spec.KudoOperatorTaskSpec.OperatorVersion)
		if err != nil {
			return fmt.Errorf(
				"failed to resolve package %s, dependency of package %s: %v", fullyQualifiedName(childTask.Spec.KudoOperatorTaskSpec), parent.OperatorVersion.FullyQualifiedName(), err)
		}

		// after resolving the dependency we update the KudoOperatorTask.Spec definition with the resolved
		// operator name, version and app version so that a definition like:
		// ---
		//   - name: deploy-child
		//    kind: KudoOperator
		//    spec:
		//      package: "../child-operator"
		//
		// will be updated as:
		// ---
		//   - name: deploy-child
		//    kind: KudoOperator
		//    spec:
		//      appVersion: 3.2.1
		//      operatorVersion: 0.0.1
		//      package: child
		//
		// so that the KUDO controller to be able to grab the right 'OperatorVersion' resources from the cluster
		// when the task is executed.
		err = updateResolvedKudoOperatorTask(childTask.Name,
			parent.OperatorVersion,
			childResolved.Resources.Operator.Name,
			childResolved.Resources.OperatorVersion.Spec.Version,
			childResolved.Resources.OperatorVersion.Spec.AppVersion)
		if err != nil {
			return err
		}

		childDependency := Dependency{
			Resources: *childResolved.Resources,
		}

		newPackage := false
		childIndex := indexOf(dependencies, &childDependency)
		if childIndex == -1 {
			clog.V(2).Printf("Adding new dependency %s", childResolved.Resources.OperatorVersion.FullyQualifiedName())
			newPackage = true

			*dependencies = append(*dependencies, childDependency)
			childIndex = len(*dependencies) - 1

			// The number of vertices in 'g' has to match the number of packages we're tracking.
			g.AddVertex()
		}

		// This is a directed graph. The edge represents a dependency of the parent package on the current package.
		g.AddEdge(parentIndex, childIndex)

		if !graph.Acyclic(g) {
			return fmt.Errorf(
				"cyclic package dependency found when adding package %s -> %s", parent.OperatorVersion.FullyQualifiedName(), childResolved.Resources.OperatorVersion.FullyQualifiedName())
		}

		// We only need to walk the dependencies if the package is new
		if newPackage {
			if err := dependencyWalk(dependencies, g, childResolved.Resources, childIndex, childResolved.DependenciesResolver); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateResolvedKudoOperatorTask method updates all 'KudoOperatorTasks' of an OperatorVersion by setting their 'Package' and
// 'OperatorVersion' fields to the already resolved packages. This is done for the KUDO controller to be able to grab
// the right 'OperatorVersion' resources from the cluster when the corresponding task is executed.
func updateResolvedKudoOperatorTask(taskName string, parent *kudoapi.OperatorVersion, operatorName, operatorVersion, appVersion string) error {
	for i, tt := range parent.Spec.Tasks {
		if tt.Name == taskName {
			parent.Spec.Tasks[i].Spec.KudoOperatorTaskSpec.Package = operatorName
			parent.Spec.Tasks[i].Spec.KudoOperatorTaskSpec.OperatorVersion = operatorVersion
			parent.Spec.Tasks[i].Spec.KudoOperatorTaskSpec.AppVersion = appVersion
			return nil
		}
	}
	return fmt.Errorf("failed to update resolved task %s of the operator %s with resolved data", taskName, parent.FullyQualifiedName())
}

// indexOf method searches for the dependency in dependencies that has the same
// OperatorVersion/AppVersion (using EqualOperatorVersion method) and returns
// its index or -1 if not found.
func indexOf(dependencies *[]Dependency, dependency *Dependency) int {
	for i, d := range *dependencies {
		if d.OperatorVersion.EqualOperatorVersion(dependency.OperatorVersion) {
			return i
		}
	}
	return -1
}

// fullyQualifiedName formats a TaskSpec for human readable consumption.
func fullyQualifiedName(kt kudoapi.KudoOperatorTaskSpec) string {
	operatorVersion := kt.OperatorVersion
	if operatorVersion == "" {
		operatorVersion = "any"
	}
	appVersion := kt.AppVersion
	if appVersion == "" {
		appVersion = "any"
	}

	return fmt.Sprintf("Operator: %q, OperatorVersion: %q, AppVersion %q", kt.Package, operatorVersion, appVersion)
}
