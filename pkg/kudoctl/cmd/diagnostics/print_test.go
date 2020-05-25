package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestPrintLog (t *testing.T){}

func TestPrintError (t *testing.T){}

func TestPrintYaml (t *testing.T){}

func Test_nonFailingPrinter_printObject(t *testing.T) {

	tests := []struct {
		name      string
		o         runtime.Object
		parentDir string
		mode      printMode
		expFiles  []string
		failOn    string
	}{
		{
			name:      "kube object with dir",
			o:         &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod"},
			},
			parentDir: "root",
			mode:      ObjectWithDir,
			expFiles: []string{"root/pod_my-fancy-pod/my-fancy-pod.yaml"},
		},
		{
			name:      "kudo object with dir",
			o:         &v1beta1.Operator{
				TypeMeta:   metav1.TypeMeta{
					Kind: "Operator",
					APIVersion: "kudo.dev/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-operator"},
			},
			parentDir: "root",
			mode:      ObjectWithDir,
			expFiles: []string{"root/operator_my-fancy-operator/my-fancy-operator.yaml"},
		},
		{
			name:      "kube object as runtime object",
			o:         &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod"},
			},
			parentDir: "root",
			mode:      RuntimeObject,
			expFiles: []string{"root/pod.yaml"},
		},
		{
			name:      "list of objects as runtime object",
			o:         &v1.PodList{
				Items:    []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
				},
			},
			parentDir: "root",
			mode:      RuntimeObject,
			expFiles: []string{"root/podlist.yaml"},
		},
		{
			name:      "list of objects with dirs",
			o:         &v1.PodList{
				Items:    []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
				},
			},
			parentDir: "root",
			mode:      ObjectListWithDirs,
			expFiles: []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml", "root/pod_my-fancy-pod-02/my-fancy-pod-02.yaml"},
		},
		{
			name:      "list of objects with dirs, one fails",
			o:         &v1.PodList{
				Items:    []v1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-01"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-02"}},
					{ObjectMeta: metav1.ObjectMeta{Namespace: fakeNamespace, Name: "my-fancy-pod-03"}},
				},
			},
			parentDir: "root",
			mode:      ObjectListWithDirs,
			expFiles:  []string{"root/pod_my-fancy-pod-01/my-fancy-pod-01.yaml", "root/pod_my-fancy-pod-03/my-fancy-pod-03.yaml"},
			failOn:    "root/pod_my-fancy-pod-02/my-fancy-pod-02.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := tt.failOn != ""
			var fs afero.Fs = afero.NewMemMapFs()
			if wantErr {
				fs = &failingFs{failOn:tt.failOn, Fs: fs}
			}
			p := &nonFailingPrinter{
				fs:    fs,
			}
			p.printObject(tt.o, tt.parentDir, tt.mode)
				for _, fname := range tt.expFiles {
					exists, _ := afero.Exists(fs, fname)
					assert.True(t, exists)
				}
			assert.Equal(t, !wantErr, len(p.errors) == 0)
		})
	}
}