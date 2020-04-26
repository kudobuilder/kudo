module github.com/kudobuilder/kudo

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/google/go-cmp v0.3.0
	github.com/gosuri/uitable v0.0.4
	github.com/kudobuilder/kuttl v0.1.0
	github.com/manifoldco/promptui v0.6.0
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.7
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/thoas/go-funk v0.6.0
	github.com/xlab/treeprint v1.0.0
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200408040146-ea54a3c99b9b // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.3
	k8s.io/cli-runtime v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/code-generator v0.17.3
	k8s.io/component-base v0.17.3
	k8s.io/kubectl v0.17.3
	sigs.k8s.io/controller-runtime v0.5.1
	sigs.k8s.io/controller-tools v0.2.6
	sigs.k8s.io/yaml v1.1.0
)

replace k8s.io/code-generator v0.17.3 => github.com/kudobuilder/code-generator v0.17.4-beta.0.0.20200316162450-cc91a9201457
