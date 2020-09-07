package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestPlanReferenceVerifier(t *testing.T) {

	resources := []string{"sally.yaml"}
	tasks := []kudoapi.Task{{
		Name: "thingOne",
		Kind: "Apply",
		Spec: kudoapi.TaskSpec{
			ResourceTaskSpec: kudoapi.ResourceTaskSpec{Resources: resources},
		},
	}}

	steps := []kudoapi.Step{{
		Name:  "cat-in-hat",
		Tasks: []string{"thingOne"},
	}}

	phases := []kudoapi.Phase{{
		Name:     "parents leave",
		Strategy: "serial",
		Steps:    steps,
	}}

	plans := make(map[string]kudoapi.Plan)
	plans["deploy"] = kudoapi.Plan{
		Strategy: "serial",
		Phases:   phases,
	}
	plans["used-plan"] = kudoapi.Plan{
		Strategy: "serial",
		Phases:   phases,
	}
	plans["unused-plan"] = kudoapi.Plan{
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
