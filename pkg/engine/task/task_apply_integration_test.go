// +build integration

package task

import (
	"fmt"
	"log"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

var testenv testutils.TestEnvironment

func TestMain(m *testing.M) {
	var err error

	testenv, err = StartTestEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()
	testenv.Environment.Stop()
	os.Exit(exitCode)
}


func TestApplyTask_Run(t *testing.T) {
	meta := renderer.Metadata{
		Metadata: engine.Metadata{
			InstanceName:        "test",
			InstanceNamespace:   "default",
			OperatorName:        "first-operator",
			OperatorVersionName: "first-operator-1.0",
			OperatorVersion:     "1.0",
		},
		PlanName:  "plan",
		PhaseName: "phase",
		StepName:  "step",
		TaskName:  "task",
	}

	tests := []struct {
		name    string
		task    ApplyTask
		done    bool
		wantErr bool
		fatal   bool
		ctx     Context
	}{
		{
			name: "effectively removes a field",
			task: ApplyTask{
				Name: "task",
				Resources:[]string{"job"},
			},
			done:true,
			wantErr:false,
			ctx: Context{
				Client: testenv.Client,
				Enhancer: &testEnhancer{},
				Meta: meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.task.Run(tt.ctx)
		assert.True(t, tt.done == got, fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))
		if tt.wantErr {
			assert.True(t, errors.Is(err, engine.ErrFatalExecution) == tt.fatal)
			assert.Error(t, err)
		}
		if !tt.wantErr {
			assert.NoError(t, err)
		}
	}
}

func pod(includeSecondContainer bool) *corev1.Pod { //nolint:unparam
	containers := []corev1.Container{
		corev1.Container{
			Name:  "container1",
			Image: "image",
		},
	}
	if includeSecondContainer {
		containers = append(containers, corev1.Container{
			Name: "container2",
			Image: "image",
		})

	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
	return pod
}

