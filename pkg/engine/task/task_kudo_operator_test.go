package task

import (
	"testing"

	"github.com/kudobuilder/kuttl/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/apis"
	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
)

func Test_applyInstance(t *testing.T) {

	operatorName := "test-operator"
	operatorVersionName := "test-0.1.0"
	namespace := "default"

	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	operatorVersion := &kudoapi.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "test-operator", Namespace: namespace},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec:       kudoapi.OperatorVersionSpec{Version: "0.1.0"},
	}

	instance, err := instanceResource("test-instance", operatorName, operatorVersionName, namespace, nil, operatorVersion, scheme)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		new     *kudoapi.Instance
		ns      string
		c       client.Client
		wantErr bool
		subset  map[string]interface{}
	}{
		{
			name:    "creating a brand new instance is successful",
			new:     instance,
			ns:      namespace,
			c:       fake.NewFakeClientWithScheme(scheme, operatorVersion),
			wantErr: false,
		},
		{
			name:    "patching an existing instance with the same spec is successful",
			new:     instance,
			ns:      namespace,
			c:       fake.NewFakeClientWithScheme(scheme, operatorVersion, instance),
			wantErr: false,
		},
		{
			name: "patching an existing instance with the updated spec is successful",
			new: func() *kudoapi.Instance {
				c := instance.DeepCopy()
				c.Spec.Parameters = map[string]string{"foo": "bar"}
				return c
			}(),
			ns:      namespace,
			c:       fake.NewFakeClientWithScheme(scheme, operatorVersion, instance),
			wantErr: false,
			subset: map[string]interface{}{
				"spec": map[string]interface{}{
					"parameters": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
		},
		{
			name: "upgrading to a new operator version works",
			new: func() *kudoapi.Instance {
				c := instance.DeepCopy()
				c.Spec.OperatorVersion = corev1.ObjectReference{
					Name: "test-0.2.0",
				}
				return c
			}(),
			ns:      namespace,
			c:       fake.NewFakeClientWithScheme(scheme, instance),
			wantErr: false,
			subset: map[string]interface{}{
				"spec": map[string]interface{}{
					"operatorVersion": map[string]interface{}{
						"name": "test-0.2.0",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := applyInstance(tt.new, tt.ns, tt.c)
			assert.True(t, (err != nil) == tt.wantErr, "applyInstance() error = %v, wantErr %v", err, tt.wantErr)

			if tt.subset != nil {
				actual, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.new)
				assert.NoError(t, err)

				err = utils.IsSubset(tt.subset, actual)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_instanceParameters(t *testing.T) {
	templates := map[string]string{
		"bb-params.yaml": `
PASSWORD: {{ .Params.BB_PASSWORD }}
`,
	}

	meta := renderer.Metadata{
		Metadata: engine.Metadata{
			InstanceName:        "test-instance",
			InstanceNamespace:   "default",
			OperatorName:        "test-operator",
			OperatorVersionName: "test-0.1.0",
			OperatorVersion:     "0.1.0",
			AppVersion:          "3.1.2",
		},
	}

	parameters := map[string]interface{}{
		"BB_PASSWORD": "secret",
	}

	tests := []struct {
		name       string
		pf         string
		templates  map[string]string
		meta       renderer.Metadata
		parameters map[string]interface{}
		want       map[string]string
		wantErr    bool
	}{
		{
			name:       "no parameter file returns an empty map",
			pf:         "",
			templates:  templates,
			meta:       meta,
			parameters: parameters,
			want:       map[string]string{},
			wantErr:    false,
		},
		{
			name:       "a parameter file populates the returned map",
			pf:         "bb-params.yaml",
			templates:  templates,
			meta:       meta,
			parameters: parameters,
			want:       map[string]string{"PASSWORD": "secret"},
			wantErr:    false,
		},
		{
			name:       "missing parameter file returns an error",
			pf:         "fake-params.yaml",
			templates:  templates,
			meta:       meta,
			parameters: parameters,
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "missing parameter returns an error",
			pf:         "bb-params.yaml",
			templates:  templates,
			meta:       meta,
			parameters: map[string]interface{}{},
			want:       nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := instanceParameters(tt.pf, tt.templates, tt.meta, tt.parameters)

			assert.True(t, (err != nil) == tt.wantErr, "instanceParameters() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, tt.want, got, "instanceParameters() got = %v, want %v", got, tt.want)
		})
	}
}
