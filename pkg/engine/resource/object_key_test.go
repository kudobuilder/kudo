package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/kudobuilder/kudo/pkg/test/fake"
)

func Test_isNamespaced(t *testing.T) {
	fdc := fake.CachedDiscoveryClient()

	tests := []struct {
		name    string
		gvk     schema.GroupVersionKind
		di      discovery.CachedDiscoveryInterface
		want    bool
		wantErr bool
	}{
		{
			name:    "pod",
			gvk:     schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			di:      fdc,
			want:    true,
			wantErr: false,
		},
		{
			name:    "namespace",
			gvk:     schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
			di:      fdc,
			want:    false,
			wantErr: false,
		},
		{
			name:    "customresourcedefinition",
			gvk:     schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"},
			di:      fdc,
			want:    false,
			wantErr: false,
		},
		{
			name:    "fake",
			gvk:     schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Fake"},
			di:      fdc,
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := isNamespaced(tt.gvk, tt.di)

			assert.True(t, (err != nil) == tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
