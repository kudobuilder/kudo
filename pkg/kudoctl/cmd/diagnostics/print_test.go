package diagnostics

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	pod1Yaml = `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: my-fancy-pod-01
  namespace: my-namespace
spec:
  containers:
  - name: my-fancy-container-01
    resources: {}
status: {}
`
	pod2Yaml = `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: my-fancy-pod-02
  namespace: my-namespace
spec:
  containers:
  - name: my-fancy-container-01
    resources: {}
status: {}
`
	podListYaml = `apiVersion: v1
items:
- metadata:
    creationTimestamp: null
    name: my-fancy-pod-01
    namespace: my-namespace
  spec:
    containers:
    - name: my-fancy-container-01
      resources: {}
  status: {}
- metadata:
    creationTimestamp: null
    name: my-fancy-pod-02
    namespace: my-namespace
  spec:
    containers:
    - name: my-fancy-container-01
      resources: {}
  status: {}
kind: PodList
metadata: {}
`
	operatorYaml = `apiVersion: kudo.dev/v1beta1
kind: Operator
metadata:
  creationTimestamp: null
  name: my-fancy-operator
  namespace: my-namespace
spec: {}
status: {}
`
	versionYaml = `gitversion: dev
gitcommit: dev
builddate: "1970-01-01T00:00:00Z"
goversion: go1.13.4
compiler: gc
platform: linux/amd64
kubernetesclientversion: v0.18.6
`
)

var (
	pod1 = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{Name: "my-fancy-container-01"}},
		},
	}
	pod2 = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{Name: "my-fancy-container-01"}},
		},
	}
	pod3 = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-03"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{Name: "my-fancy-container-01"}},
		},
	}
)

func TestPrinter_printObject(t *testing.T) {
	tests := []struct {
		desc      string
		obj       runtime.Object
		parentDir string
		mode      printMode
		expFiles  []string
		expData   []string
		failOn    string
	}{
		{
			desc:      "kube object with dir",
			obj:       &pod1,
			parentDir: "root",
			mode:      ObjectWithDir,
			expData:   []string{pod1Yaml},
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml"},
		},
		{
			desc: "kudo object with dir",
			obj: &kudoapi.Operator{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Operator",
					APIVersion: "kudo.dev/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-operator"},
			},
			parentDir: "root",
			mode:      ObjectWithDir,
			expData:   []string{operatorYaml},
			expFiles:  []string{"root/operator_my-fancy-operator/my-fancy-operator.yaml"},
		},
		{
			desc:      "kube object as runtime object",
			obj:       &pod1,
			parentDir: "root",
			mode:      RuntimeObject,
			expData:   []string{pod1Yaml},
			expFiles:  []string{"root/pod.yaml"},
		},
		{
			desc: "list of objects as runtime object",
			obj: &v1.PodList{
				Items: []v1.Pod{
					pod1,
					pod2,
				},
			},
			parentDir: "root",
			mode:      RuntimeObject,
			expData:   []string{podListYaml},
			expFiles:  []string{"root/podlist.yaml"},
		},
		{
			desc: "list of objects with dirs",
			obj: &v1.PodList{
				Items: []v1.Pod{
					pod1,
					pod2,
				},
			},
			parentDir: "root",
			mode:      ObjectListWithDirs,
			expData:   []string{pod1Yaml, pod2Yaml},
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml", "root/pod_my-fancy-pod-02/my-fancy-pod-02.yaml"},
		},
		{
			desc: "list of objects with dirs, one fails",
			obj: &v1.PodList{
				Items: []v1.Pod{
					pod1,
					pod2,
					pod3,
				},
			},
			parentDir: "root",
			mode:      ObjectListWithDirs,
			expData:   []string{pod1Yaml, pod2Yaml},
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml", "root/pod_my-fancy-pod-02/my-fancy-pod-02.yaml"},
			failOn:    "root/pod_my-fancy-pod-03/my-fancy-pod-03.yaml",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			wantErr := tt.failOn != ""
			var fs = afero.NewMemMapFs()
			if wantErr {
				fs = &failingFs{failOn: tt.failOn, Fs: fs}
			}
			p := &nonFailingPrinter{
				fs: fs,
			}
			p.printObject(tt.obj, tt.parentDir, tt.mode)
			for i, fname := range tt.expFiles {
				b, err := afero.ReadFile(fs, fname)
				assert.Nil(t, err)
				assert.Equal(t, tt.expData[i], string(b))
			}
			assert.Equal(t, !wantErr, len(p.errors) == 0)
		})
	}
}

func TestPrinter_printError(t *testing.T) {
	tests := []struct {
		desc      string
		e         error
		parentDir string
		name      string
		expFiles  []string
		expData   []string
		failOn    string
	}{
		{
			desc:      "print error OK",
			e:         errFakeTestError,
			parentDir: "root",
			name:      "service",
			expFiles:  []string{"root/service.err"},
			expData:   []string{errFakeTestError.Error()},
		},
		{
			desc:      "print error failure",
			e:         errFakeTestError,
			parentDir: "root",
			name:      "service",
			failOn:    "root/service.err",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			wantErr := tt.failOn != ""
			var fs = afero.NewMemMapFs()
			if wantErr {
				fs = &failingFs{failOn: tt.failOn, Fs: fs}
			}
			p := &nonFailingPrinter{
				fs: fs,
			}
			p.printError(tt.e, tt.parentDir, tt.name)
			for i, fname := range tt.expFiles {
				b, err := afero.ReadFile(fs, fname)
				assert.Nil(t, err)
				assert.Equal(t, tt.expData[i], string(b))
			}
			assert.Equal(t, !wantErr, len(p.errors) == 0)
		})
	}
}

func TestPrinter_printLog(t *testing.T) {
	tests := []struct {
		desc          string
		log           io.ReadCloser
		parentDir     string
		podName       string
		containerName string
		expFiles      []string
		expData       []string
		failOn        string
	}{
		{
			desc:          "print log OK",
			log:           ioutil.NopCloser(strings.NewReader(testLog)),
			parentDir:     "root",
			podName:       "my-fancy-pod-01",
			containerName: "my-fancy-container-01",
			expFiles:      []string{"root/pod_my-fancy-pod-01/my-fancy-container-01.log.gz"},
			expData:       []string{testLogGZipped},
		},
		{
			desc:          "print log failure",
			log:           ioutil.NopCloser(strings.NewReader(testLog)),
			parentDir:     "root",
			podName:       "my-fancy-pod-01",
			containerName: "my-fancy-container-01",
			failOn:        "root/pod_my-fancy-pod-01/my-fancy-container-01.log.gz",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			wantErr := tt.failOn != ""
			var fs = afero.NewMemMapFs()
			if wantErr {
				fs = &failingFs{failOn: tt.failOn, Fs: fs}
			}
			p := &nonFailingPrinter{
				fs: fs,
			}
			p.printLog(tt.log, filepath.Join(tt.parentDir, fmt.Sprintf("pod_%s", tt.podName)), tt.containerName)
			for i, fname := range tt.expFiles {
				b, err := afero.ReadFile(fs, fname)
				assert.Nil(t, err)
				assert.Equal(t, tt.expData[i], string(b))
			}
			assert.Equal(t, !wantErr, len(p.errors) == 0)
		})
	}
}

func TestPrinter_printYaml(t *testing.T) {
	tests := []struct {
		desc      string
		v         interface{}
		parentDir string
		name      string
		expFiles  []string
		expData   []string
		failOn    string
	}{
		{
			desc: "print Yaml OK",
			v: version.Info{
				GitVersion:              "dev",
				GitCommit:               "dev",
				BuildDate:               "1970-01-01T00:00:00Z",
				GoVersion:               "go1.13.4",
				Compiler:                "gc",
				Platform:                "linux/amd64",
				KubernetesClientVersion: "v0.18.6",
			},
			parentDir: "root",
			name:      "version",
			expFiles:  []string{"root/version.yaml"},
			expData:   []string{versionYaml},
			failOn:    "",
		},
		{
			desc:      "print Yaml OK",
			v:         version.Info{},
			parentDir: "root",
			name:      "version",
			failOn:    "root/version.yaml",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			wantErr := tt.failOn != ""
			var fs = afero.NewMemMapFs()
			if wantErr {
				fs = &failingFs{failOn: tt.failOn, Fs: fs}
			}
			p := &nonFailingPrinter{
				fs: fs,
			}
			p.printYaml(tt.v, tt.parentDir, tt.name)
			for i, fname := range tt.expFiles {
				b, err := afero.ReadFile(fs, fname)
				assert.Nil(t, err)
				assert.Equal(t, tt.expData[i], string(b))
			}
			assert.Equal(t, !wantErr, len(p.errors) == 0)
		})
	}
}
