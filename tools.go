// +build tools

// Package tools is used to import go modules that we use for tooling as dependencies.
// For more information, please refer to: https://github.com/go-modules-by-example/index/blob/ac9bf72/010_tools/README.md
package tools

import (
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
