package instance

import (
	"reflect"
	"testing"

	"github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/pkg/errors"

	"github.com/ghodss/yaml"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExecutePlan(t *testing.T) {
	defaultMetadata := &executionMetadata{
		instanceName:        "Instance",
		planExecutionID:     "pid",
		instanceNamespace:   "default",
		operatorVersion:     "ov-1.0",
		operatorName:        "operator",
		resourcesOwner:      getJob("pod2", "default"),
		operatorVersionName: "ovname",
		appVersion:          "3.4.10-test_version",
	}
	tests := []struct {
		name           string
		activePlan     *activePlan
		metadata       *executionMetadata
		expectedStatus *v1alpha1.PlanStatus
	}{
		{"plan already finished", &activePlan{
			Name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionComplete,
			},
		}, defaultMetadata, &v1alpha1.PlanStatus{
			Status: v1alpha1.ExecutionComplete,
		}},
		{"plan with one step to be executed, still in progress", &activePlan{
			Name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionPending,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionPending, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionPending, Name: "step"}}}},
			},
			Spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks:     map[string]v1alpha1.TaskSpec{"task": {Resources: []string{"job"}}},
			Templates: map[string]string{"job": getResourceAsString(getJob("job1", "default"))},
		}, defaultMetadata, &v1alpha1.PlanStatus{
			Status: v1alpha1.ExecutionInProgress,
			Name:   "test",
			Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionInProgress, Name: "step"}}}},
		}},
		// this plan deploys pod, that is marked as healthy immediately because we cannot evaluate health
		{"plan with one step, immediately healthy -> completed", &activePlan{
			Name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionPending,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionPending, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionPending, Name: "step"}}}},
			},
			Spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks:     map[string]v1alpha1.TaskSpec{"task": {Resources: []string{"pod"}}},
			Templates: map[string]string{"pod": getResourceAsString(getPod("pod1", "default"))},
		}, defaultMetadata, &v1alpha1.PlanStatus{
			Status: v1alpha1.ExecutionComplete,
			Name:   "test",
			Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
		}},
		{"plan in errored state will be retried and completed when no error happens", &activePlan{
			Name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ErrorStatus,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ErrorStatus, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ErrorStatus, Name: "step"}}}},
			},
			Spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks:     map[string]v1alpha1.TaskSpec{"task": {Resources: []string{"pod"}}},
			Templates: map[string]string{"pod": getResourceAsString(getPod("pod1", "default"))},
		}, defaultMetadata, &v1alpha1.PlanStatus{
			Status: v1alpha1.ExecutionComplete,
			Name:   "test",
			Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
		}},
	}

	for _, tt := range tests {
		testClient := fake.NewFakeClientWithScheme(scheme.Scheme)
		newStatus, err := executePlan(tt.activePlan, tt.metadata, testClient, &testKubernetesObjectEnhancer{})

		if err != nil {
			t.Errorf("%s: Expecting no error but got error %v", tt.name, err)
		}

		if !reflect.DeepEqual(tt.expectedStatus, newStatus) {
			t.Errorf("%s: Expecting status to be %v but got %v", tt.name, *tt.expectedStatus, *newStatus)
		}
	}
}

func getJob(name string, namespace string) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{},
	}
	return job
}

func getPod(name string, namespace string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{},
	}
	return pod
}

func getResourceAsString(resource v1.Object) string {
	bytes, _ := yaml.Marshal(resource)
	return string(bytes)
}

type testKubernetesObjectEnhancer struct{}

func (k *testKubernetesObjectEnhancer) applyConventionsToTemplates(templates map[string]string, metadata metadata, owner v1.Object) ([]runtime.Object, error) {
	result := make([]runtime.Object, 0)
	for _, t := range templates {
		objsToAdd, err := template.ParseKubernetesObjects(t)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing kubernetes objects after applying kustomize")
		}
		result = append(result, objsToAdd[0])
	}
	return result, nil
}
