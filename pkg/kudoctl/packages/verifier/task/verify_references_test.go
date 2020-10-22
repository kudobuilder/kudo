package task

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTaskReferenceVerifier(t *testing.T) {

	// 2 task , 1 referenced, 1 not referenced (results in warning)
	resources := []string{"sally.yaml"}
	tasks := []kudoapi.Task{{
		Name: "thingOne",
		Kind: "Apply",
		Spec: kudoapi.TaskSpec{
			ResourceTaskSpec: kudoapi.ResourceTaskSpec{Resources: resources},
		},
	}, {
		Name: "thingTwo",
		Kind: "Apply",
		Spec: kudoapi.TaskSpec{
			ResourceTaskSpec: kudoapi.ResourceTaskSpec{Resources: resources},
		},
	}}

	steps := []kudoapi.Step{{
		Name:  "cat-in-hat",
		Tasks: []string{"thingOne"},
	}, {
		Name:  "mayham",
		Tasks: []string{"thingThree"},
	}}

	phases := []kudoapi.Phase{{
		Name:     "parents leave",
		Strategy: "serial",
		Steps:    steps,
	}}

	plans := make(map[string]kudoapi.Plan)
	plans["boring-rainy"] = kudoapi.Plan{
		Strategy: "serial",
		Phases:   phases,
	}

	operator := packages.OperatorFile{
		Tasks: tasks,
		Plans: plans,
	}

	pf := packages.Files{
		Operator: &operator,
	}
	verifier := ReferenceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 1, len(res.Warnings))
	assert.Equal(t, `task "thingTwo" defined but not used`, res.Warnings[0])
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, `task "thingThree" in plan "boring-rainy" is not defined`, res.Errors[0])
}
