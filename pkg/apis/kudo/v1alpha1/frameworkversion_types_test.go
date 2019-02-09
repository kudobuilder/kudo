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

package v1alpha1

import (
	"fmt"
	"gopkg.in/go-playground/validator.v9"
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageFrameworkVersion(t *testing.T) {
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &FrameworkVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &FrameworkVersion{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

//func TestFrameworkType_Validation(t *testing.T) {
//
//	emptyString := ""
//	something := "SOMETHING"
//	taskVolumeType := "UNKNOWN"
//
//	strategy := "parallel"
//
//	taskList := map[string]*Task{
//		"": {},
//	}
//
//	taskPortEnvKey := ""
//	taskPort := map[string]*Port{
//		"": {},
//		"http-api": {
//			Port:      65536,
//			EnvKey:    &taskPortEnvKey,
//			Advertise: true,
//		},
//	}
//
//	serviceSpec := ServiceSpec{
//		Name: &emptyString,
//		Scheduler: &Scheduler{
//			Principal: &emptyString,
//			Zookeeper: &emptyString,
//			User:      &emptyString,
//		},
//		Pods: map[string]*Pod{
//			"": {},
//			"pod2": {
//				ResourceSets: map[string]*ResourceSet{
//					"": {},
//					"shared": {
//						Cpus:     -1,
//						Gpus:     -1,
//						MemoryMB: -2,
//						Ports:    taskPort,
//					},
//				},
//				Placement: &emptyString,
//				Count:     -1,
//				Image:     &emptyString,
//				Networks: map[string]*Network{
//					"": {},
//					"dcos": {
//						HostPorts: []int32{
//							0,
//							1,
//							1,
//							65536,
//						},
//						ContainerPorts: []int32{
//							-1,
//						},
//					},
//				},
//				RLimits: map[string]*RLimit{
//					"": {},
//					"RLIMIT_NOFILE": {
//						Soft: 0,
//						Hard: 0,
//					},
//				},
//				Uris: []*string{
//					&emptyString,
//					&something,
//				},
//				Tasks: taskList,
//				Volume: &Volume{
//					Path:   &emptyString,
//					Type:   &taskVolumeType,
//					SizeMB: -1,
//				},
//				Volumes: map[string]*Volume{
//					"": {
//						Type: &emptyString,
//					},
//					"opt-mesosphere": {},
//				},
//				PreReservedRole: &emptyString,
//				Secrets: map[string]*Secret{
//					"": {},
//					"keytab": {
//						SecretPath: &emptyString,
//					},
//					"SECRET2": {
//						SecretPath: &emptyString,
//						EnvKey:     &emptyString,
//						FilePath:   &emptyString,
//					},
//				},
//				HostVolumes: map[string]*HostVolume{
//					"": {},
//					"opt-mesosphere": {
//						HostPath:      &emptyString,
//						ContainerPath: &emptyString,
//					},
//				},
//			},
//			"pod3": {},
//		},
//		Plans: map[string]*Plan{
//			"": {},
//			"plan1": {
//				Strategy: &strategy,
//			},
//			"plan2": {
//				Strategy: &strategy,
//				Phases: map[string]*Phase{
//					"": {},
//					"phase1": {
//						Strategy: &something,
//						Steps: []*map[string]*[][]string{
//							{},
//							{
//								"": {},
//								"default": {
//									{
//										"",
//										"main",
//									},
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//
//	// TaskSpec
//	taskEnvValue := "1000"
//	taskGoal := "ONCE2"
//	taskCmd := ""
//
//	taskEnv := map[string]*string{
//		"ENV_VARIABLE": &taskEnvValue,
//		"":             &taskEnvValue,
//		"NIL":          nil,
//	}
//
//	//taskConfigTemplate := "krb5.conf"
//	taskConfig := map[string]*Config{
//		"": nil,
//		"krb5-conf": {
//			Template: nil,
//			Dest:     nil,
//		},
//	}
//
//	taskHealthCheckCmd := ""
//
//	taskHealthCheck := HealthCheck{
//		Cmd: &taskHealthCheckCmd,
//		//Interval: -1,
//		GracePeriodSecs:        -1,
//		MaxConsecutiveFailures: -1,
//		DelaySecs:              -1,
//		TimeoutSecs:            -1,
//	}
//
//	taskReadinessCheck := ReadinessCheck{
//		Cmd:          &taskHealthCheckCmd,
//		IntervalSecs: -1,
//		DelaySecs:    -1,
//		TimeoutSecs:  -1,
//	}
//
//	taskSpec := Task{
//		Goal:           &taskGoal,
//		Essential:      true,
//		Cmd:            &taskCmd,
//		Env:            taskEnv,
//		Configs:        taskConfig,
//		Cpus:           -0.1,
//		Gpus:           -1,
//		MemoryMB:       -2,
//		Ports:          taskPort,
//		HealthCheck:    &taskHealthCheck,
//		ReadinessCheck: &taskReadinessCheck,
//		Volume: &Volume{
//			Path:   &emptyString,
//			Type:   &taskVolumeType,
//			SizeMB: -1,
//		},
//		Volumes: map[string]*Volume{
//			"": {
//				Type: &emptyString,
//			},
//			"opt-mesosphere": {},
//		},
//		TaskKillGracePeriodSecs: -1,
//		TransportEncryption: []*TransportEncryption{
//			{},
//			{
//				Name: &emptyString,
//				Type: &taskVolumeType,
//			},
//		},
//	}
//
//	expectedErrors := []string{
//		"ServiceSpec.Name:gt: cannot be empty",
//		"ServiceSpec.WebUrl:required: cannot be <nil>",
//		"ServiceSpec.Scheduler.Principal:gt: cannot be empty",
//		"ServiceSpec.Scheduler.Zookeeper:gt: cannot be empty",
//		"ServiceSpec.Scheduler.User:gt: cannot be empty",
//		"ServiceSpec.Pods[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].ResourceSets[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Cpus:gt: cannot be -1",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Gpus:gt: cannot be -1",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].MemoryMB:gte: cannot be -2",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Ports[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Ports[].Port:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Ports[http-api].Port:max: cannot be 65536",
//		"ServiceSpec.Pods[pod2].ResourceSets[shared].Ports[http-api].EnvKey:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Placement:gt: cannot be empty",
//		"ServiceSpec.Pods[].Count:gte: cannot be 0",
//		"ServiceSpec.Pods[].Tasks:required: cannot be map[]",
//		"ServiceSpec.Pods[pod2].Count:gte: cannot be -1",
//		"ServiceSpec.Pods[pod3].Count:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].Image:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Networks[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].Networks[].HostPorts:gte: cannot be []",
//		"ServiceSpec.Pods[pod2].Networks[].ContainerPorts:gte: cannot be []",
//		"ServiceSpec.Pods[pod2].Networks[dcos].HostPorts[0]:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].Networks[dcos].HostPorts[3]:max: cannot be 65536",
//		"ServiceSpec.Pods[pod2].Networks[dcos].ContainerPorts[0]:gte: cannot be -1",
//		"ServiceSpec.Pods[pod2].RLimits[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].RLimits[].Soft:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].RLimits[].Hard:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].RLimits[RLIMIT_NOFILE].Soft:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].RLimits[RLIMIT_NOFILE].Hard:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].Uris[0]:gte: cannot be empty",
//		"ServiceSpec.Pods[pod2].Tasks[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].Tasks[].Goal:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].Tasks[].Cmd:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod3].Tasks:required: cannot be map[]",
//		"ServiceSpec.Pods[pod2].Volume.Path:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Volume.Type:eq=ROOT|eq=MOUNT: cannot be UNKNOWN",
//		"ServiceSpec.Pods[pod2].Volume.SizeMB:gte: cannot be -1",
//		"ServiceSpec.Pods[pod2].Volumes[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].Volumes[].Path:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].Volumes[].Type:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Volumes[].SizeMB:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].Volumes[opt-mesosphere].Path:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].Volumes[opt-mesosphere].Type:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].Volumes[opt-mesosphere].SizeMB:gte: cannot be 0",
//		"ServiceSpec.Pods[pod2].PreReservedRole:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Secrets[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].Secrets[].SecretPath:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].Secrets[keytab].SecretPath:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Secrets[SECRET2].SecretPath:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Secrets[SECRET2].EnvKey:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].Secrets[SECRET2].FilePath:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].HostVolumes[opt-mesosphere].HostPath:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].HostVolumes[opt-mesosphere].ContainerPath:gt: cannot be empty",
//		"ServiceSpec.Pods[pod2].HostVolumes[]:required: cannot be empty",
//		"ServiceSpec.Pods[pod2].HostVolumes[].HostPath:required: cannot be <nil>",
//		"ServiceSpec.Pods[pod2].HostVolumes[].ContainerPath:required: cannot be <nil>",
//		"ServiceSpec.Plans[]:required: cannot be empty",
//		"ServiceSpec.Plans[].Strategy:required: cannot be <nil>",
//		"ServiceSpec.Plans[].Phases:gt: cannot be map[]",
//		"ServiceSpec.Plans[plan1].Phases:gt: cannot be map[]",
//		"ServiceSpec.Plans[plan2].Phases[]:required: cannot be empty",
//		"ServiceSpec.Plans[plan2].Phases[].Strategy:required: cannot be <nil>",
//		"ServiceSpec.Plans[plan2].Phases[].Pod:required: cannot be <nil>",
//		"ServiceSpec.Plans[plan2].Phases[phase1].Steps[0]:gte: cannot be map[]",
//		"ServiceSpec.Plans[plan2].Phases[phase1].Steps[1][]:gt: cannot be []",
//		"ServiceSpec.Plans[plan2].Phases[phase1].Pod:required: cannot be <nil>",
//	}
//
//	expectedTaskErrors := []string{
//		"Task.Goal:eq=RUNNING|eq=FINISH|eq=ONCE: cannot be ONCE2",
//		"Task.Essential:isdefault: cannot be true",
//		"Task.Cmd:gt: cannot be empty",
//		"Task.Env[NIL]:required: cannot be <nil>",
//		"Task.Env[]:required: cannot be empty",
//		"Task.Configs[]:required: cannot be empty",
//		"Task.Configs[]:required: cannot be <nil>",
//		"Task.Configs[krb5-conf].Template:required: cannot be <nil>",
//		"Task.Configs[krb5-conf].Dest:required: cannot be <nil>",
//		"Task.Cpus:gt: cannot be -0.1",
//		"Task.Gpus:gt: cannot be -1",
//		"Task.MemoryMB:gte: cannot be -2",
//		"Task.Ports[]:required: cannot be empty",
//		"Task.Ports[].Port:gte: cannot be 0",
//		"Task.Ports[http-api].Port:max: cannot be 65536",
//		"Task.Ports[http-api].EnvKey:gt: cannot be empty",
//		"Task.HealthCheck.Cmd:gt: cannot be empty",
//		"Task.HealthCheck.GracePeriodSecs:gte: cannot be -1",
//		"Task.HealthCheck.MaxConsecutiveFailures:gte: cannot be -1",
//		"Task.HealthCheck.DelaySecs:gte: cannot be -1",
//		"Task.HealthCheck.TimeoutSecs:gte: cannot be -1",
//		"Task.ReadinessCheck.Cmd:gt: cannot be empty",
//		"Task.ReadinessCheck.IntervalSecs:gte: cannot be -1",
//		"Task.ReadinessCheck.DelaySecs:gte: cannot be -1",
//		"Task.ReadinessCheck.TimeoutSecs:gte: cannot be -1",
//		"Task.Volume.Path:gt: cannot be empty",
//		"Task.Volume.Type:eq=ROOT|eq=MOUNT: cannot be UNKNOWN",
//		"Task.Volume.SizeMB:gte: cannot be -1",
//		"Task.Volumes[]:required: cannot be empty",
//		"Task.Volumes[].Path:required: cannot be <nil>",
//		"Task.Volumes[].Type:gt: cannot be empty",
//		"Task.Volumes[].SizeMB:gte: cannot be 0",
//		"Task.Volumes[opt-mesosphere].Path:required: cannot be <nil>",
//		"Task.Volumes[opt-mesosphere].Type:required: cannot be <nil>",
//		"Task.Volumes[opt-mesosphere].SizeMB:gte: cannot be 0",
//		"Task.TaskKillGracePeriodSecs:gte: cannot be -1",
//		"Task.TransportEncryption[0].Name:required: cannot be <nil>",
//		"Task.TransportEncryption[0].Type:required: cannot be <nil>",
//		"Task.TransportEncryption[1].Name:gt: cannot be empty",
//		"Task.TransportEncryption[1].Type:eq=TLS|eq=KEYSTORE: cannot be UNKNOWN",
//	}
//
//	validate := validator.New()
//
//	tests := []struct {
//		in  interface{}
//		err []string
//	}{
//		{serviceSpec, expectedErrors},  // 1
//		{taskSpec, expectedTaskErrors}, // 2
//	}
//
//	for i, tt := range tests {
//		i := i
//		vErr := validate.Struct(tt.in)
//		if vErr != nil {
//			_, ok := vErr.(validator.ValidationErrors)
//			if !ok {
//				t.Errorf("%d: expected ValidationError got: %v", i+1, vErr)
//				continue
//			}
//
//			receivedErrorList := createSlice(vErr)
//
//			diff := compareSlice(receivedErrorList, tt.err)
//			if diff != nil {
//				for _, err := range diff {
//					t.Errorf("Found unexpected error: %v", err)
//				}
//			}
//			missing := compareSlice(tt.err, receivedErrorList)
//			if missing != nil {
//				for _, err := range missing {
//					t.Errorf("Missed expected error: %v", err)
//				}
//			}
//		}
//	}
//}

func createSlice(e error) []string {

	receivedErrorList := []string{}

	for _, err := range e.(validator.ValidationErrors) {
		empty := err.Value()
		if err.Value() == "" {
			empty = "empty"
		}
		receivedError := fmt.Sprintf("%v:%v: cannot be %v", err.Namespace(), err.Tag(), empty)
		receivedErrorList = append(receivedErrorList, receivedError)
	}
	return receivedErrorList
}

func compareSlice(real, mock []string) []string {
	lm := len(mock)

	var diff []string = nil

	for _, rv := range real {
		i := 0
		j := 0
		for _, mv := range mock {
			i++
			if rv == mv {
				continue
			}
			if rv != mv {
				j++
			}
			if lm <= j {
				diff = append(diff, rv)
			}
		}
	}
	return diff
}
