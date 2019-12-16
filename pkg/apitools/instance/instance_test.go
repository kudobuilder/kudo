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

package instance

import (
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetLastExecutedPlanStatus(t *testing.T) {
	testTime := time.Date(
		2019, 10, 17, 1, 1, 1, 1, time.UTC)
	tests := []struct {
		name             string
		planStatus       map[string]v1beta1.PlanStatus
		expectedPlanName string
	}{
		{"no plan ever run", map[string]v1beta1.PlanStatus{"test": {
			Status: v1beta1.ExecutionNeverRun,
			Name:   "test",
			Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionNeverRun, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionNeverRun, Name: "step"}}}},
		}}, ""},
		{"plan in progress", map[string]v1beta1.PlanStatus{
			"test": {
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionInProgress, Name: "step"}}}},
			},
			"test2": {
				Status: v1beta1.ExecutionComplete,
				Name:   "test2",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionComplete, Name: "step"}}}},
			}}, "test"},
		{"last executed plan", map[string]v1beta1.PlanStatus{
			"test": {
				Status:          v1beta1.ExecutionComplete,
				Name:            "test",
				LastFinishedRun: v1.Time{Time: testTime},
				Phases:          []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status:          v1beta1.ExecutionComplete,
				Name:            "test2",
				LastFinishedRun: v1.Time{Time: testTime.Add(time.Hour)},
				Phases:          []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionComplete, Name: "step"}}}},
			}}, "test2"},
	}

	for _, tt := range tests {
		i := v1beta1.Instance{}
		i.Status.PlanStatus = tt.planStatus
		actual := GetLastExecutedPlanStatus(&i)
		actualName := ""
		if actual != nil {
			actualName = actual.Name
		}
		if actualName != tt.expectedPlanName {
			t.Errorf("%s: Expected to get plan %s but got plan status of %v", tt.name, tt.expectedPlanName, actual)
		}
	}
}
