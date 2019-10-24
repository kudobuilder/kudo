module github.com/kudobuilder/kudo

go 1.13

require (
	cloud.google.com/go v0.38.0
	contrib.go.opencensus.io/exporter/ocagent v0.4.12 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/containerd/containerd v1.2.9 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20190916154449-92cc603036dd
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustinkirkland/golang-petname v0.0.0-20170921220637-d3c2ba80e75e
	github.com/ghodss/yaml v1.0.0
	github.com/go-test/deep v1.0.1
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golangci/gocyclo v0.0.0-20180528144436-0a533e8fa43d // indirect
	github.com/golangci/golangci-lint v1.21.0 // indirect
	github.com/golangci/revgrep v0.0.0-20180812185044-276a5c0a1039 // indirect
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/gophercloud/gophercloud v0.2.0 // indirect
	github.com/gostaticanalysis/analysisutil v0.0.3 // indirect
	github.com/gosuri/uitable v0.0.3
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/magiconair/properties v1.8.1
	github.com/masterminds/sprig v2.18.0+incompatible
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pelletier/go-toml v1.5.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/securego/gosec v0.0.0-20191008095658-28c1128b7336 // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/uudashr/gocognit v1.0.0 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478
	golang.org/x/sys v0.0.0-20191024073052-e66fe6eb8e0c // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools v0.0.0-20191024074452-7defa796fec0
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	gopkg.in/yaml.v2 v2.2.4
	honnef.co/go/tools v0.0.1-2019.2.3
	k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/klog v1.0.0 // indirect
	mvdan.cc/unparam v0.0.0-20190917161559-b83a221c10a2 // indirect
	sigs.k8s.io/controller-runtime v0.3.1-0.20191022174215-ad57a976ffa1
	sigs.k8s.io/controller-tools v0.2.0
	sigs.k8s.io/kind v0.5.1
	sigs.k8s.io/kustomize v2.0.3+incompatible
	sigs.k8s.io/yaml v1.1.0
	sourcegraph.com/sqs/pbtypes v1.0.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
