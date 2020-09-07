package task

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTaskBasicsVerifier(t *testing.T) {
	tests := []struct {
		name     string
		task     kudoapi.Task
		errors   []string
		warnings []string
	}{
		{name: "An Apply task without resources", task: kudoapi.Task{
			Name: "Task",
			Kind: "Apply",
			Spec: kudoapi.TaskSpec{},
		}, errors: []string{"task validation error: apply task 'Task' has an empty resource list. if that's what you need, use a Dummy task instead"}, warnings: []string{}},
		{name: "An Apply task with resources", task: kudoapi.Task{
			Name: "Task",
			Kind: "Apply",
			Spec: kudoapi.TaskSpec{
				ResourceTaskSpec: kudoapi.ResourceTaskSpec{Resources: []string{"someResource"}},
			},
		}, errors: []string{}, warnings: []string{}},
		{name: "An Delete task without resources", task: kudoapi.Task{
			Name: "Task",
			Kind: "Delete",
			Spec: kudoapi.TaskSpec{},
		}, errors: []string{"task validation error: delete task 'Task' has an empty resource list. if that's what you need, use a Dummy task instead"}, warnings: []string{}},
		{name: "An Delete task with resources", task: kudoapi.Task{
			Name: "Task",
			Kind: "Delete",
			Spec: kudoapi.TaskSpec{
				ResourceTaskSpec: kudoapi.ResourceTaskSpec{Resources: []string{"someResource"}},
			},
		}, errors: []string{}, warnings: []string{}},

		// More detailed tests are in engine/task/task_test.go
		{name: "An empty pipe task", task: kudoapi.Task{
			Name: "Task",
			Kind: "Pipe",
			Spec: kudoapi.TaskSpec{},
		}, errors: []string{"task validation error: pipe task has an empty pipe files list"}, warnings: []string{}},
		{name: "A valid pipe task", task: kudoapi.Task{
			Name: "Task",
			Kind: "Pipe",
			Spec: kudoapi.TaskSpec{
				PipeTaskSpec: kudoapi.PipeTaskSpec{
					Pod: "",
					Pipe: []kudoapi.PipeSpec{
						{
							Kind: "ConfigMap",
							File: "someFile",
							Key:  "someKey",
						},
					},
				},
			},
		}, errors: []string{}, warnings: []string{}},
	}
	verifier := BuildVerifier{}

	for _, test := range tests {
		test := test
		pf := packageFilesFromWithTask(test.task)
		res := verifier.Verify(&pf)

		assert.Equal(t, test.errors, res.Errors, test.name)
		assert.Equal(t, test.warnings, res.Warnings, test.name)
	}
}

func packageFilesFromWithTask(task kudoapi.Task) packages.Files {
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
		Tasks: []kudoapi.Task{task},
		Plans: plans,
	}

	return packages.Files{
		Operator: &operator,
	}
}
