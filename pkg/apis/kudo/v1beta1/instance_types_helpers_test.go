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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var (
	testTime = time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)
	testUUID = uuid.NewUUID()
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

func TestInstance_ResetPlanStatus(t *testing.T) {
	// a test instance with 'deploy' plan IN_PROGRESS and 'update' that NEVER_RUN
	instance := &Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "foo-operator",
			},
			Parameters: map[string]string{
				"foo": "foo",
			},
		},
		Status: InstanceStatus{
			PlanStatus: map[string]PlanStatus{
				"deploy": {
					Status:               ExecutionInProgress,
					Name:                 "deploy",
					LastUpdatedTimestamp: &metav1.Time{Time: testTime},
					UID:                  testUUID,
					Phases:               []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
				},
				"update": {
					Status: ExecutionNeverRun,
					Name:   "update",
					Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
				},
			},
			AggregatedStatus: AggregatedStatus{
				Status:         ExecutionInProgress,
				ActivePlanName: "deploy",
			},
		},
	}

	oldUID := instance.Status.PlanStatus["deploy"].UID

	err := instance.ResetPlanStatus("deploy", &metav1.Time{Time: time.Now()})
	assert.NoError(t, err)

	// we test that UID has changed. afterwards, we replace it with the old one and compare new
	// plan status with the desired state
	assert.NotEqual(t, instance.Status.PlanStatus["deploy"].UID, oldUID)
	oldPlanStatus := instance.Status.PlanStatus["deploy"]
	statusCopy := oldPlanStatus.DeepCopy()
	statusCopy.UID = testUUID
	statusCopy.LastUpdatedTimestamp = &metav1.Time{Time: testTime}
	instance.Status.PlanStatus["deploy"] = *statusCopy

	// Expected:
	// - the status of the 'deploy' plan to be reset: all phases and steps should be PENDING, new UID should be assigned
	// - AggregatedStatus should be PENDING too and 'deploy' should be the active plan
	// - 'update' plan status should be unchanged
	assert.Equal(t, InstanceStatus{
		PlanStatus: map[string]PlanStatus{
			"deploy": {
				Status:               ExecutionPending,
				Name:                 "deploy",
				LastUpdatedTimestamp: &metav1.Time{Time: testTime},
				UID:                  testUUID,
				Phases:               []PhaseStatus{{Name: "phase", Status: ExecutionPending, Steps: []StepStatus{{Status: ExecutionPending, Name: "step"}}}},
			},
			"update": {
				Status: ExecutionNeverRun,
				Name:   "update",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
			},
		},
		AggregatedStatus: AggregatedStatus{
			Status:         ExecutionPending,
			ActivePlanName: "deploy",
		},
	}, instance.Status)
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
