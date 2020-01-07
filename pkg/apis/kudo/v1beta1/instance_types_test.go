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

	"github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetLastExecutedPlanStatus(t *testing.T) {
	testTime := time.Date(
		2019, 10, 17, 1, 1, 1, 1, time.UTC)
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
				LastFinishedRun: v1.Time{Time: testTime},
				Phases:          []PhaseStatus{{Name: "phase", Status: ExecutionComplete, Steps: []StepStatus{{Status: ExecutionComplete, Name: "step"}}}},
			},
			"test2": {
				Status:          ExecutionComplete,
				Name:            "test2",
				LastFinishedRun: v1.Time{Time: testTime.Add(time.Hour)},
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

func TestSpecParameterDifference(t *testing.T) {
	var testParams = []struct {
		name string
		new  map[string]string
		diff map[string]string
	}{
		{"update one value", map[string]string{"one": "11", "two": "2"}, map[string]string{"one": "11"}},
		{"update multiple values", map[string]string{"one": "11", "two": "22"}, map[string]string{"one": "11", "two": "22"}},
		{"add new value", map[string]string{"one": "1", "two": "2", "three": "3"}, map[string]string{"three": "3"}},
		{"remove one value", map[string]string{"one": "1"}, map[string]string{"two": "2"}},
		{"no difference", map[string]string{"one": "1", "two": "2"}, map[string]string{}},
		{"empty new map", map[string]string{}, map[string]string{"one": "1", "two": "2"}},
	}

	g := gomega.NewGomegaWithT(t)

	var old = map[string]string{"one": "1", "two": "2"}

	for _, test := range testParams {
		diff := parameterDiff(old, test.new)
		g.Expect(diff).Should(gomega.Equal(test.diff), test.name)
	}
}
