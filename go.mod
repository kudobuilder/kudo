module github.com/kudobuilder/kudo

go 1.13

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/docker/docker v1.4.2-0.20190916154449-92cc603036dd
	github.com/dustinkirkland/golang-petname v0.0.0-20191129215211-8e5a1ed0cff0
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gophercloud/gophercloud v0.2.0 // indirect
	github.com/gosuri/uitable v0.0.4
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kudobuilder/kuttl v0.0.0-20200323171915-a03c0f1bc3b4
	github.com/manifoldco/promptui v0.6.0
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/thoas/go-funk v0.5.0
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/code-generator v0.17.3
	k8s.io/component-base v0.17.3
	k8s.io/kubectl v0.17.3
	sigs.k8s.io/controller-runtime v0.5.1
	sigs.k8s.io/controller-tools v0.2.6
	sigs.k8s.io/kind v0.6.1
	sigs.k8s.io/yaml v1.1.0
)

replace k8s.io/code-generator v0.17.3 => github.com/kudobuilder/code-generator v0.17.4-beta.0.0.20200316162450-cc91a9201457
