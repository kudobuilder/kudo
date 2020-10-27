package resolver

import (
	"strings"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// PackageResolver is the source of resolver of operator packages.
type PackageResolver struct {
	local *LocalHelper
	uri   *URLHelper
	repo  *repo.Client
}

// NewPackageResolver creates an operator package resolver for non-repository packages
func NewPackageResolver(repo *repo.Client) packages.Resolver {
	return &PackageResolver{
		local: NewLocalHelper(),
		uri:   NewURLHelper(),
		repo:  repo,
	}
}

func (m *PackageResolver) copyWithFs(fs afero.Fs) packages.Resolver {
	out := m
	out.local = NewForFilesystem(fs)
	return out
}

// NewInClusterResolver returns an initialized InClusterResolver for resolving already installed packages
func NewInClusterResolver(c *kudo.Client, ns string) packages.Resolver {
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
func (m *PackageResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.PackageScope, error) {

	clog.V(2).Printf("determining package type of %v", name)

	// 1. local files/folder have priority
	abs, err := m.local.LocalPackagePath(name)

	// LocalPackagePath returns an error if name isn't a local file/folder and does not indicate other errors
	if err == nil {
		clog.V(2).Printf("local operator discovered: %v", abs)

		var res *packages.Resources
		if strings.HasSuffix(abs, ".tgz") {
			out := afero.NewMemMapFs()
			res, err = m.local.ResolveTar(out, abs)
			if err == nil {
				return &packages.PackageScope{
					Resources:            res,
					DependenciesResolver: m.copyWithFs(out),
					OperatorDirectory:    "/"}, nil
			}
		} else {
			res, err = m.local.ResolveDir(abs)
			if err == nil {
				return &packages.PackageScope{
					Resources:            res,
					DependenciesResolver: m,
					OperatorDirectory:    abs}, nil
			}
		}

		return nil, err
	}

	// 2. next are tarball URLs
	clog.V(3).Printf("no local operator discovered, looking for http")
	if http.IsValidURL(name) {
		clog.V(3).Printf("operator using http protocol for %v", name)
		out := afero.NewMemMapFs()
		res, err := m.uri.ResolveURL(out, name)
		if err == nil {
			return &packages.PackageScope{
				Resources:            res,
				DependenciesResolver: m.copyWithFs(out),
				OperatorDirectory:    "/"}, nil
		}
		return nil, err
	}

	// 3. try the repo as the last
	clog.V(3).Printf("no http discovered, looking for repository")
	out := afero.NewMemMapFs()
	res, err := m.repo.Resolve(out, name, appVersion, operatorVersion)
	if err == nil {
		return &packages.PackageScope{
			Resources:            res,
			DependenciesResolver: m.copyWithFs(out),
			OperatorDirectory:    "/"}, nil
	}
	return nil, err
}
