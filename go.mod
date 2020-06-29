module github.com/kudobuilder/kudo

go 1.14

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/google/go-cmp v0.4.0
	github.com/gosuri/uitable v0.0.4
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/kudobuilder/kuttl v0.4.0
	github.com/manifoldco/promptui v0.7.0
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/thoas/go-funk v0.6.0
	github.com/xlab/treeprint v1.0.0
	github.com/yourbasic/graph v0.0.0-20170921192928-40eb135c0b26
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200408040146-ea54a3c99b9b // indirect
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/cli-runtime v0.18.4
	k8s.io/client-go v0.18.4
	k8s.io/code-generator v0.18.4
	k8s.io/component-base v0.18.4
	k8s.io/kubectl v0.18.4
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

// Replace this when kuttl 0.5.0 is released
replace github.com/kudobuilder/kuttl v0.4.0 => github.com/kudobuilder/kuttl v0.4.1-0.20200626203555-914c2ca0a2b5
