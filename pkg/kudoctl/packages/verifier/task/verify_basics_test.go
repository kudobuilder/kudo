package task

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTaskBasicsVerifier(t *testing.T) {
	tests := []struct {
		name     string
		task     v1beta1.Task
		errors   []string
		warnings []string
	}{
		{name: "An Apply task without resources", task: v1beta1.Task{
			Name: "Task",
			Kind: "Apply",
			Spec: v1beta1.TaskSpec{},
		}, errors: []string{"task validation error: apply task 'Task' has an empty resource list. if that's what you need, use a Dummy task instead"}, warnings: []string{}},
		{name: "An Apply task with resources", task: v1beta1.Task{
			Name: "Task",
			Kind: "Apply",
			Spec: v1beta1.TaskSpec{
				ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: []string{"someResource"}},
			},
		}, errors: []string{}, warnings: []string{}},
		{name: "An Delete task without resources", task: v1beta1.Task{
			Name: "Task",
			Kind: "Delete",
			Spec: v1beta1.TaskSpec{},
		}, errors: []string{"task validation error: delete task 'Task' has an empty resource list. if that's what you need, use a Dummy task instead"}, warnings: []string{}},
		{name: "An Delete task with resources", task: v1beta1.Task{
			Name: "Task",
			Kind: "Delete",
			Spec: v1beta1.TaskSpec{
				ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: []string{"someResource"}},
			},
		}, errors: []string{}, warnings: []string{}},

		// More detailed tests are in engine/task/task_test.go
		{name: "An empty pipe task", task: v1beta1.Task{
			Name: "Task",
			Kind: "Pipe",
			Spec: v1beta1.TaskSpec{},
		}, errors: []string{"task validation error: pipe task has an empty pipe files list"}, warnings: []string{}},
		{name: "A valid pipe task", task: v1beta1.Task{
			Name: "Task",
			Kind: "Pipe",
			Spec: v1beta1.TaskSpec{
				PipeTaskSpec: v1beta1.PipeTaskSpec{
					Pod: "",
					Pipe: []v1beta1.PipeSpec{
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
	verifier := BasicVerifier{}

	for _, test := range tests {
		test := test
		pf := packageFilesFromWithTask(test.task)
		res := verifier.Verify(&pf)

		assert.Equal(t, test.errors, res.Errors, test.name)
		assert.Equal(t, test.warnings, res.Warnings, test.name)
	}
}

func packageFilesFromWithTask(task v1beta1.Task) packages.Files {
	steps := []v1beta1.Step{{
		Name:  "cat-in-hat",
		Tasks: []string{"thingOne"},
	}, {
		Name:  "mayham",
		Tasks: []string{"thingThree"},
	}}

	phases := []v1beta1.Phase{{
		Name:     "parents leave",
		Strategy: "serial",
		Steps:    steps,
	}}

	plans := make(map[string]v1beta1.Plan)
	plans["boring-rainy"] = v1beta1.Plan{
		Strategy: "serial",
		Phases:   phases,
	}

	operator := packages.OperatorFile{
		Tasks: []v1beta1.Task{task},
		Plans: plans,
	}

	return packages.Files{
		Operator: &operator,
	}
}
