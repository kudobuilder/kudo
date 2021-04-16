package convert

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
)

var update = flag.Bool("update", false, "update .golden files")

func Test_JsonSchemaExport(t *testing.T) {

	tests := []struct {
		name       string
		ov         *kudoapi.OperatorVersion
		outputType output.Type
		err        string
	}{
		{name: "simple.json", ov: simpleOv(), outputType: output.TypeJSON},
		{name: "simple.yaml", ov: simpleOv(), outputType: output.TypeYAML},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			err := WriteJSONSchema(tt.ov, tt.outputType, out)
			assert.NoError(t, err)

			rendered := out.String()

			gf := fmt.Sprintf("testdata/%s.golden", tt.name)

			if *update {
				t.Log("update golden file")

				//nolint:gosec
				if err := ioutil.WriteFile(gf, []byte(rendered), 0644); err != nil {
					t.Fatalf("failed to update golden file: %s", err)
				}
			}

			golden, err := ioutil.ReadFile(gf)
			if err != nil {
				t.Fatalf("failed reading .golden: %s", err)
			}

			assert.Equal(t, string(golden), rendered, "for golden file: %s", gf)
		})
	}

}

func simpleOv() *kudoapi.OperatorVersion {
	return &kudoapi.OperatorVersion{
		ObjectMeta: v1.ObjectMeta{
			Name: "Test",
		},
		Spec: kudoapi.OperatorVersionSpec{
			Groups: []kudoapi.ParameterGroup{
				{
					Name:        "simplegroup",
					DisplayName: "SimpleGroup",
					Description: "The description for this group.",
					Priority:    10,
				},
			},
			Parameters: []kudoapi.Parameter{
				{
					Name:        "my-param",
					DisplayName: "MyParam",
					Description: "My parameter description",
					Type:        kudoapi.StringValueType,
					Immutable:   boolPtr(true),
					Default:     strPointer("defaultVal"),
					Hint:        "My Param Hint.",
					Trigger:     "my-plan",
				},
				{
					Name:        "grouped-param",
					DisplayName: "GroupedParam",
					Description: "My parameter description in a group",
					Type:        kudoapi.IntegerValueType,
					Immutable:   boolPtr(false),
					Default:     strPointer("23"),
					Hint:        "My Group Param Hint.",
					Group:       "simplegroup",
				},
			},
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}
func strPointer(v string) *string {
	return &v
}
