package resolver

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// Resolver will try to resolve a given package name to either local tarball, folder, remote url or
// an operator in the remote repository.
type Resolver interface {
	Resolve(name string, version string) (*packages.Package, error)
}

// PackageResolver is the source of resolver of operator packages.
type PackageResolver struct {
	local *LocalFinder
	uri   *URLFinder
	repo  *repo.Client
}

// New creates an operator package resolver for non-repository packages
func New(repo *repo.Client) *PackageResolver {
	lf := NewLocal()
	uf := NewURL()
	return &PackageResolver{
		local: lf,
		uri:   uf,
		repo:  repo,
	}
}

// Resolve provides a one stop to acquire any non-repo packages by trying to look for package files
// resolving the operator name to:
// - a local tgz file
// - a local directory
// - a url to a tgz
// - an operator name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
func (m *PackageResolver) Resolve(name string, version string) (*packages.Package, error) {

	// Local files/folder have priority
	if _, err := m.local.fs.Stat(name); err == nil {
		clog.V(2).Printf("local operator discovered: %v", name)
		b, err := m.local.Resolve(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	clog.V(3).Printf("no local operator discovered, looking for http")
	if http.IsValidURL(name) {
		clog.V(3).Printf("operator using http protocol for %v", name)
		b, err := m.uri.Resolve(name, version)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	clog.V(3).Printf("no http discovered, looking for repository")
	if b, err := m.repo.Resolve(name, version); err == nil {
		return b, nil
	}

	return nil, fmt.Errorf("resolver: unable to find packages for %v", name)
}
