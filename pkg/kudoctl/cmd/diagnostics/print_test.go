package diagnostics

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
  containers: null
status: {}
`
	pod2Yaml = `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: my-fancy-pod-02
  namespace: my-namespace
spec:
  containers: null
status: {}
`
	podListYaml = `apiVersion: v1
items:
- metadata:
    creationTimestamp: null
    name: my-fancy-pod-01
    namespace: my-namespace
  spec:
    containers: null
  status: {}
- metadata:
    creationTimestamp: null
    name: my-fancy-pod-02
    namespace: my-namespace
  spec:
    containers: null
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
`
)

func TestPrintLog(t *testing.T) {}

func TestPrintError(t *testing.T) {}

func TestPrintYaml(t *testing.T) {}

func TestPrinter_printObject(t *testing.T) {
	tests := []struct {
		desc      string
		o         runtime.Object
		parentDir string
		mode      printMode
		expFiles  []string
		expData   []string
		failOn    string
	}{
		{
			desc: "kube object with dir",
			o: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"},
			},
			parentDir: "root",
			mode:      ObjectWithDir,
			expData:   []string{pod1Yaml},
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml"},
		},
		{
			desc: "kudo object with dir",
			o: &v1beta1.Operator{
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
			desc: "kube object as runtime object",
			o: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"},
			},
			parentDir: "root",
			mode:      RuntimeObject,
			expData:   []string{pod1Yaml},
			expFiles:  []string{"root/pod.yaml"},
		},
		{
			desc: "list of objects as runtime object",
			o: &v1.PodList{
				Items: []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
				},
			},
			parentDir: "root",
			mode:      RuntimeObject,
			expData:   []string{podListYaml},
			expFiles:  []string{"root/podlist.yaml"},
		},
		{
			desc: "list of objects with dirs",
			o: &v1.PodList{
				Items: []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
				},
			},
			parentDir: "root",
			mode:      ObjectListWithDirs,
			expData:   []string{pod1Yaml, pod2Yaml},
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml", "root/pod_my-fancy-pod-02/my-fancy-pod-02.yaml"},
		},
		{
			desc: "list of objects with dirs, one fails",
			o: &v1.PodList{
				Items: []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-03"}},
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
			p.printObject(tt.o, tt.parentDir, tt.mode)
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
		desc      string
		log       io.ReadCloser
		parentDir string
		podName   string
		expFiles  []string
		expData   []string
		failOn    string
	}{
		{
			desc:      "print log OK",
			log:       ioutil.NopCloser(strings.NewReader("Ein Fichtenbaum steht einsam im Norden auf kahler Höh")),
			parentDir: "root",
			podName:   "my-fancy-pod-01",
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.log.gz"},
			expData: []string{"\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xffr\xcd\xccSp\xcbL\xce(I\xcdKJ,\xcdU(.I\xcd(QH\xcd" +
				"\xcc+N\xccU\xc8\xccU\xf0\xcb/JI\xcdSH,MS\xc8N\xcc\xc8I-R\xf08\xbc-\x03\x10\x00\x00\xff\xff\x13\xa1nx6\x00\x00\x00"},
		},
		{
			desc:      "print log failure",
			log:       ioutil.NopCloser(strings.NewReader("Ein Fichtenbaum steht einsam im Norden auf kahler Höh")),
			parentDir: "root",
			podName:   "my-fancy-pod-01",
			failOn:    "root/pod_my-fancy-pod-01/my-fancy-pod-01.log.gz",
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
			p.printLog(tt.log, tt.parentDir, tt.podName)
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
				GitVersion: "dev",
				GitCommit:  "dev",
				BuildDate:  "1970-01-01T00:00:00Z",
				GoVersion:  "go1.13.4",
				Compiler:   "gc",
				Platform:   "linux/amd64",
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
