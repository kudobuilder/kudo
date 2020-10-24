package resolver

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

func TestInClusterResolver_Resolve(t *testing.T) {
	testOperator := &kudoapi.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-operator",
			Namespace: "default",
		},
		Spec: kudoapi.OperatorSpec{
			Description: "A foo Operator",
			KudoVersion: "0.16.0",
		},
	}

	testOperatorVersion := func(operatorVersion, appVersion string) *kudoapi.OperatorVersion {
		return &kudoapi.OperatorVersion{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1beta1",
				Kind:       "OperatorVersion",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kudoapi.OperatorVersionName("foo-operator", appVersion, operatorVersion),
				Namespace: "default",
			},
			Spec: kudoapi.OperatorVersionSpec{
				Operator: v1.ObjectReference{
					APIVersion: "kudo.dev/v1beta1",
					Kind:       "Operator",
					Name:       "foo-operator",
				},
				Version:    operatorVersion,
				AppVersion: appVersion,
			},
		}
	}

	c := kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
	r := InClusterResolver{c: c, ns: "default"}

	// Init the fake client with an operator and three operator versions:
	_, err := c.InstallOperatorObjToCluster(testOperator, "default")
	if err != nil {
		log.Fatal(err)
	}
	_, err = c.InstallOperatorVersionObjToCluster(testOperatorVersion("0.1.0", "3.12.1"), "default")
	if err != nil {
		log.Fatal(err)
	}
	_, err = c.InstallOperatorVersionObjToCluster(testOperatorVersion("0.2.0", "3.13.2"), "default")
	if err != nil {
		log.Fatal(err)
	}
	_, err = c.InstallOperatorVersionObjToCluster(testOperatorVersion("0.3.0", ""), "default")
	if err != nil {
		log.Fatal(err)
	}

	tests := []struct {
		name                string
		operatorName        string
		appVersion          string
		operatorVersion     string
		wantOperatorVersion string
		wantAppVersion      string
		wantErr             bool
	}{
		{
			name:                "resolve by only operator name returns version 0.2.0",
			operatorName:        "foo-operator",
			appVersion:          "",
			operatorVersion:     "",
			wantOperatorVersion: "0.2.0", // not 0.3.0 because operators *with* appVersion have priority
			wantAppVersion:      "3.13.2",
			wantErr:             false,
		},
		{
			name:                "resolve by operator version returns the right version",
			operatorName:        "foo-operator",
			appVersion:          "",
			operatorVersion:     "0.3.0",
			wantOperatorVersion: "0.3.0",
			wantAppVersion:      "",
			wantErr:             false,
		},
		{
			name:                "resolve only by the app version returns the right version",
			operatorName:        "foo-operator",
			appVersion:          "3.13.2",
			operatorVersion:     "",
			wantOperatorVersion: "0.2.0",
			wantAppVersion:      "3.13.2",
			wantErr:             false,
		},
		{
			name:                "resolve by the app AND operator version returns the right version",
			operatorName:        "foo-operator",
			appVersion:          "3.13.2",
			operatorVersion:     "0.2.0",
			wantOperatorVersion: "0.2.0",
			wantAppVersion:      "3.13.2",
			wantErr:             false,
		},
		{
			name:                "resolve by the wrong app version returns an error",
			operatorName:        "foo-operator",
			appVersion:          "3.15.0",
			operatorVersion:     "",
			wantOperatorVersion: "",
			wantAppVersion:      "",
			wantErr:             true,
		},
		{
			name:                "resolve by the wrong operator version returns an error",
			operatorName:        "foo-operator",
			appVersion:          "",
			operatorVersion:     "15.0.0",
			wantOperatorVersion: "",
			wantAppVersion:      "",
			wantErr:             true,
		},
		{
			name:                "resolve the wrong operator name",
			operatorName:        "bar-operator",
			appVersion:          "",
			operatorVersion:     "",
			wantOperatorVersion: "",
			wantAppVersion:      "",
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.operatorName, tt.appVersion, tt.operatorVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Errorf("Resolve() failed to resolve an operator foo with operatorVersion=%s and appVersion=%s", tt.wantOperatorVersion, tt.wantAppVersion)
					return
				}

				assert.Equal(t, got.Resources.OperatorVersion.Spec.Version, tt.wantOperatorVersion, "got: %v", got.Resources.OperatorVersion)
			}
		})
	}

}
