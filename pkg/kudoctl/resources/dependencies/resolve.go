package dependencies

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/thoas/go-funk"
	"github.com/yourbasic/graph"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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

type Dependency struct {
	packages.Resources

	PackageName string
}

// Resolve resolves all dependencies of an OperatorVersion. Dependencies are resolved recursively.
// Cyclic dependencies are detected and result in an error. operatorArgument parameter is string that
// the user passed to install/upgrade an operator which is used to determine whether this is a local
// operator in a directory. In that case all its relative dependencies (if any exist) will be relative
// to the operator directory (the one with `operator.yaml` file).
// See github.com/kudobuilder/kudo/issues/1701 for additional context.
func Resolve(operatorArgument string, operatorVersion *kudoapi.OperatorVersion, resolver pkgresolver.Resolver) ([]Dependency, error) {
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

	operatorDir, _ := operatorAbsPath(operatorArgument)

	if err := dependencyWalk(&dependencies, &g, root, 0, resolver, operatorDir); err != nil {
		return nil, err
	}

	// Remove 'root' from the list of dependencies.
	return dependencies[1:], nil
}

func dependencyWalk(
	dependencies *[]Dependency,
	g *dependencyGraph,
	parent packages.Resources,
	parentIndex int,
	resolver pkgresolver.Resolver,
	operatorDirectory *string) error {
	//nolint:errcheck
	childrenTasks := funk.Filter(parent.OperatorVersion.Spec.Tasks, func(task kudoapi.Task) bool {
		return task.Kind == engtask.KudoOperatorTaskKind
	}).([]kudoapi.Task)

	for _, childTask := range childrenTasks {
		childPackageName := childTask.Spec.KudoOperatorTaskSpec.Package

		// if the path to a child dependency is a relative one, we construct the absolute path for the
		// resolver by combining the absolute path for the operator directory with the relative dependency path
		// a relative dependency path must begin with either './' or '../'
		isRelativePackage := strings.HasPrefix(childPackageName, "./") || strings.HasPrefix(childPackageName, "../")
		if isRelativePackage {
			if operatorDirectory == nil {
				return fmt.Errorf("a dependency with a relative path %q is only allowed in a parent %v when it is a local directory", childPackageName, parent.OperatorVersion.FullyQualifiedName())
			}
			childDirectory, err := operatorAbsPath(filepath.Join(*operatorDirectory, childPackageName))
			if err != nil {
				return fmt.Errorf("local dependency %s of the parent %s has an invalid path: %v", fullyQualifiedName(childTask.Spec.KudoOperatorTaskSpec), parent.OperatorVersion.FullyQualifiedName(), err)
			}
			childPackageName = *childDirectory

		}

		childPkg, err := resolver.Resolve(
			childPackageName,
			childTask.Spec.KudoOperatorTaskSpec.AppVersion,
			childTask.Spec.KudoOperatorTaskSpec.OperatorVersion)
		if err != nil {
			return fmt.Errorf(
				"failed to resolve package %s, dependency of package %s: %v", fullyQualifiedName(childTask.Spec.KudoOperatorTaskSpec), parent.OperatorVersion.FullyQualifiedName(), err)
		}

		childDependency := Dependency{
			Resources:   *childPkg.Resources,
			PackageName: childTask.Spec.KudoOperatorTaskSpec.Package,
		}

		newPackage := false
		childIndex := indexOf(dependencies, &childDependency)
		if childIndex == -1 {
			clog.V(2).Printf("Adding new dependency %s", childPkg.Resources.OperatorVersion.FullyQualifiedName())
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
				"cyclic package dependency found when adding package %s -> %s", parent.OperatorVersion.FullyQualifiedName(), childPkg.Resources.OperatorVersion.FullyQualifiedName())
		}

		// We only need to walk the dependencies if the package is new
		if newPackage {
			var childOperatorDirectory *string
			if isRelativePackage {
				childOperatorDirectory = &childPackageName
			}
			if err := dependencyWalk(dependencies, g, *childPkg.Resources, childIndex, resolver, childOperatorDirectory); err != nil {
				return err
			}
		}
	}

	return nil
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

func fullyQualifiedName(kt kudoapi.KudoOperatorTaskSpec) string {
	return fmt.Sprintf("%s-%s", kudoapi.OperatorVersionName(kt.Package, kt.OperatorVersion), kt.AppVersion)
}

func operatorAbsPath(path string) (*string, error) {
	fs := afero.NewOsFs()

	ok, err := afero.IsDir(fs, path)
	// force local operators usage to be either absolute or express a relative path
	// or put another way, a name can NOT be mistaken to be the name of a local folder
	if ok && filepath.Base(path) != path {
		abs, err := filepath.Abs(path)
		if err == nil {
			return &abs, nil
		}
		return nil, fmt.Errorf("failed to detect an absolute path for %q: %v", path, err)
	}
	return nil, fmt.Errorf("%q is not a valid directory: %v", path, err)
}
