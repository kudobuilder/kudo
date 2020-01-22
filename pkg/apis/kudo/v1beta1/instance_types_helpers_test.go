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
		{"no plan ever run", map[string]PlanStatus{"test": {
			Status: ExecutionNeverRun,
			Name:   "test",
			Phases: []PhaseStatus{{Name: "phase", Status: ExecutionNeverRun, Steps: []StepStatus{{Status: ExecutionNeverRun, Name: "step"}}}},
		}}, ""},
		{"plan in progress", map[string]PlanStatus{
			"test": {
				Status: ExecutionInProgress,
				Name:   "test",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
			},
			"test2": {
				Status: ExecutionComplete,
				Name:   "test2",
				Phases: []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}, "test"},
		{"last executed plan", map[string]PlanStatus{
			"test": {
				Status:          ExecutionComplete,
				Name:            "test",
				LastFinishedRun: metav1.Time{Time: testTime},
				Phases:          []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status:          ExecutionComplete,
				Name:            "test2",
				LastFinishedRun: metav1.Time{Time: testTime.Add(time.Hour)},
				Phases:          []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			}}, "test2"},
	}

	for _, tt := range tests {
		i := Instance{}
		i.Status.PlanStatus = tt.planStatus
		actual := i.GetLastExecutedPlanStatus()
		actualName := ""
		if actual != nil {
			actualName = actual.Name
		}
		if actualName != tt.expectedPlanName {
			t.Errorf("%s: Expected to get plan %s but got plan status of %v", tt.name, tt.expectedPlanName, actual)
		}
	}
}

func TestInstance_ResetPlanStatus(t *testing.T) {
	// a test instance with 'deploy' plan IN_PROGRESS and 'update' that NEVER_RUN
	i := &Instance{
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
					Status:          ExecutionInProgress,
					Name:            "deploy",
					LastFinishedRun: metav1.Time{Time: testTime},
					UID:             testUUID,
					Phases:          []PhaseStatus{{Name: "phase", Status: ExecutionInProgress, Steps: []StepStatus{{Status: ExecutionInProgress, Name: "step"}}}},
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

	tests := []struct {
		name     string
		instance *Instance
		plan     string
		wantErr  bool
	}{
		{
			name:     "resetting a deploy plan updates the instance accordingly",
			instance: i,
			plan:     "deploy",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldUID := tt.instance.Status.PlanStatus["deploy"].UID
			oldUpdateStatus := tt.instance.Status.PlanStatus["update"]

			err := tt.instance.ResetPlanStatus(tt.plan)

			assert.Equal(t, tt.wantErr, err != nil)

			// check aggregated, plan, phase and step statuses
			aggregatedStatus := tt.instance.Status.AggregatedStatus
			assert.Equal(t, "deploy", aggregatedStatus.ActivePlanName)
			assert.Equal(t, ExecutionPending, aggregatedStatus.Status)
			assert.Equal(t, ExecutionPending, aggregatedStatus.Status)

			planStatus := tt.instance.Status.PlanStatus["deploy"]
			assert.NotEqual(t, planStatus.UID, oldUID)
			assert.Equal(t, ExecutionPending, planStatus.Status)

			phaseStatus := GetPhaseStatus("phase", &planStatus)
			assert.Equal(t, ExecutionPending, phaseStatus.Status)

			stepStatus := GetStepStatus("step", phaseStatus)
			assert.Equal(t, ExecutionPending, stepStatus.Status)

			// 'update' plan status should be unaffected
			assert.EqualValues(t, oldUpdateStatus, tt.instance.Status.PlanStatus["update"])
		})
	}
}
