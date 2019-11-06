package kudo

import (
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/finder"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// Resources tries to look for package files resolving the operator name to:
// - a local tgz file
// - a local directory
// - a url to a tgz
// - an operator name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
// TODO: move this to the 'packages' package -- currently not possible because of a dependency cycle
func Resources(operatorName string, version string, repository repo.Repository) (*packages.Resources, error) {
	p, err := pkgFinder(operatorName, version, repository)
	if err != nil {
		return nil, err
	}
	return p.GetCRDs()
}

// PkgFiles tries to look for package files resolving the operator name to:
// - a local tgz file
// - a local directory
// - a url to a tgz
// - an operator name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
func PkgFiles(operatorName string, version string, repository repo.Repository) (*packages.PackageFiles, error) {
	p, err := pkgFinder(operatorName, version, repository)
	if err != nil {
		return nil, err
	}
	return p.GetPkgFiles()
}

// pkgFinder provides the functionality of "hunting" for the package
func pkgFinder(operatorName string, version string, repository repo.Repository) (packages.Package, error) {
	// Local files/folder have priority
	if _, err := os.Stat(operatorName); err == nil {
		clog.V(2).Printf("local operator discovered: %v", operatorName)
		f := finder.NewLocal()
		p, err := f.GetPackage(operatorName, version)
		if err != nil {
			return nil, err
		}
		return p, nil
	}

	clog.V(3).Printf("no local operator discovered, looking for http")
	if http.IsValidURL(operatorName) {
		clog.V(3).Printf("operator using http protocol for %v", operatorName)
		f := finder.NewURL()
		p, err := f.GetPackage(operatorName, version)
		if err != nil {
			return nil, err
		}
		return p, nil
	}

	clog.V(3).Printf("no http discovered, looking for repository")
	p, err := repository.GetPackage(operatorName, version)
	if err != nil {
		return nil, err
	}
	return p, nil
}
