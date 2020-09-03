/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testTime = time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)
)

func TestGetLastExecutedPlanStatus(t *testing.T) {
	tests := []struct {
		name             string
		planStatus       map[string]PlanStatus
		expectedPlanName string
	}{
		{name: "no plan ever run", planStatus: map[string]PlanStatus{"test": {
			Status: ExecutionNeverRun,
			Name:   "test",
			Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
		}}},
		{name: "plan in progress", planStatus: map[string]PlanStatus{
			"test": {
				Status: ExecutionInProgress,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}, expectedPlanName: "test"},
		{name: "last executed plan", planStatus: map[string]PlanStatus{
			"test": {
				Status:               ExecutionComplete,
				Name:                 "test",
				LastUpdatedTimestamp: &metav1.Time{Time: testTime},
				Phases:               []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status:               ExecutionComplete,
				Name:                 "test2",
				LastUpdatedTimestamp: &metav1.Time{Time: testTime.Add(time.Hour)},
				Phases:               []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}, expectedPlanName: "test2"},
	}

	for _, tt := range tests {
		i := Instance{}
		i.Status.PlanStatus = tt.planStatus
		actual := i.GetLastExecutedPlanStatus()
		if actual != nil {
			assert.Equal(t, tt.expectedPlanName, actual.Name)
		} else {
			assert.True(t, tt.expectedPlanName == "")
		}
	}
}

func TestGetParamDefinitions(t *testing.T) {
	ov := &OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: OperatorVersionSpec{
			Parameters: []Parameter{
				{Name: "foo"},
				{Name: "other-foo"},
				{Name: "bar"},
			},
		},
	}

	tests := []struct {
		name    string
		params  map[string]string
		ov      *OperatorVersion
		want    []Parameter
		wantErr bool
	}{
		{
			name:    "all parameters exists",
			params:  map[string]string{"foo": "1", "bar": "2"},
			ov:      ov,
			want:    []Parameter{{Name: "foo"}, {Name: "bar"}},
			wantErr: false,
		},
		{
			name:    "one parameter is missing",
			params:  map[string]string{"foo": "1", "fake-one": "2"},
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "both parameters are missing",
			params:  map[string]string{"fake-one": "1", "fake-two": "2"},
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetParamDefinitions(tt.params, tt.ov)

			assert.True(t, (err != nil) == tt.wantErr, "GetParamDefinitions() error = %v, wantErr %v", err, tt.wantErr)
			assert.ElementsMatch(t, got, tt.want, "GetParamDefinitions() got = %v, want %v", got, tt.want)
		})
	}
}

func TestGetPlanInProgress(t *testing.T) {
	tests := []struct {
		name             string
		planStatus       map[string]PlanStatus
		expectedPlanName string
	}{
		{name: "no plan ever run", planStatus: map[string]PlanStatus{"test": {
			Status: ExecutionNeverRun,
			Name:   "test",
			Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
		}}},
		{name: "plan in progress", planStatus: map[string]PlanStatus{
			"test": {
				Status: ExecutionInProgress,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}, expectedPlanName: "test"},
		{name: "all plans complete", planStatus: map[string]PlanStatus{
			"test": {
				Status: ExecutionComplete,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}},
	}

	for _, tt := range tests {
		i := Instance{}
		i.Status.PlanStatus = tt.planStatus
		actual := i.GetPlanInProgress()
		if actual != nil {
			assert.Equal(t, tt.expectedPlanName, actual.Name)
		} else {
			assert.True(t, tt.expectedPlanName == "")
		}
	}
}

func TestNoPlanEverExecuted(t *testing.T) {
	tests := []struct {
		name           string
		planStatus     map[string]PlanStatus
		expectedResult bool
	}{
		{name: "no plan ever run", planStatus: map[string]PlanStatus{"test": {
			Status: ExecutionNeverRun,
			Name:   "test",
			Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
		}}, expectedResult: true},
		{name: "plan in progress", planStatus: map[string]PlanStatus{
			"test": {
				Status: ExecutionInProgress,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}},
		{name: "all plans complete", planStatus: map[string]PlanStatus{
			"test": {
				Status: ExecutionComplete,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}},
	}

	for _, tt := range tests {
		i := Instance{}
		i.Status.PlanStatus = tt.planStatus
		actual := i.NoPlanEverExecuted()
		if actual != tt.expectedResult {
			t.Errorf("%s: Expected to get plan %v but got %v", tt.name, tt.expectedResult, actual)
		}
		assert.Equal(t, tt.expectedResult, actual)
	}
}
