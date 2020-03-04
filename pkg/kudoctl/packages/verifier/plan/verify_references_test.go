package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestPlanReferenceVerifier(t *testing.T) {

	resources := []string{"sally.yaml"}
	tasks := []v1beta1.Task{{
		Name: "thingOne",
		Kind: "Apply",
		Spec: v1beta1.TaskSpec{
			ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: resources},
		},
	}}

	steps := []v1beta1.Step{{
		Name:  "cat-in-hat",
		Tasks: []string{"thingOne"},
	}}

	phases := []v1beta1.Phase{{
		Name:     "parents leave",
		Strategy: "serial",
		Steps:    steps,
	}}

	plans := make(map[string]v1beta1.Plan)
	plans["deploy"] = v1beta1.Plan{
		Strategy: "serial",
		Phases:   phases,
	}
	plans["used-plan"] = v1beta1.Plan{
		Strategy: "serial",
		Phases:   phases,
	}
	plans["unused-plan"] = v1beta1.Plan{
		Strategy: "serial",
		Phases:   phases,
	}

	operator := packages.OperatorFile{
		Tasks: tasks,
		Plans: plans,
	}
	params := packages.ParamsFile{
		Parameters: packages.Parameters{
			packages.Parameter{
				Name:    "PARAM1",
				Trigger: "used-plan",
			},
			packages.Parameter{
				Name:    "PARAM2",
				Trigger: "not-existing-plan",
			},
		},
	}

	pf := packages.Files{
		Operator: &operator,
		Params:   &params,
	}
	verifier := ReferenceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `plan "not-existing-plan" used in parameter "PARAM2" is not defined`, res.Errors[0])
}
