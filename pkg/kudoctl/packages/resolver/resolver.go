package resolver

import (
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// Resolver will try to resolve a given package name to either local tarball, folder, remote url or
// an operator in the remote repository.
type Resolver interface {
	Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error)
}

// PackageResolver is the source of resolver of operator packages.
type PackageResolver struct {
	local *LocalResolver
	uri   *URLResolver
	repo  *repo.Client
}

// New creates an operator package resolver for non-repository packages
func New(repo *repo.Client) Resolver {
	lf := NewLocal()
	uf := NewURL()
	return &PackageResolver{
		local: lf,
		uri:   uf,
		repo:  repo,
	}
}

// NewInClusterResolver returns an initialized InClusterResolver for resolving already installed packages
func NewInClusterResolver(c *kudo.Client, ns string) Resolver {
	return &InClusterResolver{c: c, ns: ns}
}

// Resolve provides a one stop to acquire any non-repo packages by trying to look for package files
// resolving the operator name to:
// - a local tgz file
// - a local directory
// - a url to a tgz
// - an operator name in the remote repository
// in that order.
// For local access there is a need to provide absolute or relative path as part of the name argument. `cassandra` without a path
// component will resolve to the remote repo.  `./cassandra` will resolve to a folder which is expected to have the operator structure on the filesystem.
// `../folder/cassandra.tgz` will resolve to the cassandra package tarball on the filesystem.
func (m *PackageResolver) Resolve(name string, appVersion string, operatorVersion string) (p *packages.Package, err error) {

	// Local files/folder have priority
	_, err = m.local.fs.Stat(name)
	// force local operators usage to be either absolute or express a relative path
	// or put another way, a name can NOT be mistaken to be the name of a local folder
	if filepath.Base(name) != name && err == nil {
		var abs string
		abs, err = filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		clog.V(2).Printf("local operator discovered: %v", abs)
		p, err = m.local.Resolve(name, appVersion, operatorVersion)
		return
	}

	clog.V(3).Printf("no local operator discovered, looking for http")
	if http.IsValidURL(name) {
		clog.V(3).Printf("operator using http protocol for %v", name)
		p, err = m.uri.Resolve(name, appVersion, operatorVersion)
		return
	}

	clog.V(3).Printf("no http discovered, looking for repository")
	p, err = m.repo.Resolve(name, appVersion, operatorVersion)

	return
}
