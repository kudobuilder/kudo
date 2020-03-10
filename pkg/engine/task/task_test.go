package task

import (
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name     string
		taskYaml string
		want     Tasker
		wantErr  bool
	}{
		{
			name: "apply task",
			taskYaml: `
name: apply-task
kind: Apply
spec: 
    resources:
      - pod.yaml
      - service.yaml`,
			want: ApplyTask{
				Name:      "apply-task",
				Resources: []string{"pod.yaml", "service.yaml"},
			},
			wantErr: false,
		},
		{
			name: "delete task",
			taskYaml: `
name: delete-task
kind: Delete
spec: 
    resources:
      - pod.yaml
      - service.yaml`,
			want: DeleteTask{
				Name:      "delete-task",
				Resources: []string{"pod.yaml", "service.yaml"},
			},
			wantErr: false,
		},
		{
			name: "dummy task",
			taskYaml: `
name: dummy-task
kind: Dummy
spec: 
    wantErr: true`,
			want: DummyTask{
				Name:    "dummy-task",
				WantErr: true,
			},
			wantErr: false,
		},
		{
			name: "pipe task with a pipe file kind Secret",
			taskYaml: `
name: pipe-task
kind: Pipe
spec:
  pod: pipe-pod.yaml
  pipe:
    - file: /tmp/foo.txt
      kind: Secret
      key: Foo`,
			want: PipeTask{
				Name: "pipe-task",
				Pod:  "pipe-pod.yaml",
				PipeFiles: []PipeFile{
					{
						File: "/tmp/foo.txt",
						Kind: "Secret",
						Key:  "Foo",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "pipe task with a pipe file kind ConfigMap",
			taskYaml: `
name: pipe-task
kind: Pipe
spec:
  pod: pipe-pod.yaml
  pipe:
    - file: /tmp/bar.txt
      kind: ConfigMap
      key: Bar`,
			want: PipeTask{
				Name: "pipe-task",
				Pod:  "pipe-pod.yaml",
				PipeFiles: []PipeFile{
					{
						File: "/tmp/bar.txt",
						Kind: "ConfigMap",
						Key:  "Bar",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "pipe task with an invalid pipe file kind",
			taskYaml: `
name: pipe-task
kind: Pipe
spec:
  pod: pipe-pod.yaml
  pipe:
    - file: /tmp/bar.txt
      kind: Invalid
      key: Bar`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "pipe task file must be defined",
			taskYaml: `
name: pipe-task
kind: Pipe
spec:
  pod: pipe-pod.yaml
  pipe:
    - file:
      kind: Secret
      key: Bar`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "pipe task key is invalid",
			taskYaml: `
name: pipe-task
kind: Pipe
spec:
  pod: pipe-pod.yaml
  pipe:
    - file: /tmp/bar.txt"
      kind: Secret
      key: $Bar^`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "unknown task",
			taskYaml: `
name: unknown-task
kind: Unknown
spec: 
    known: false`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			task := &v1beta1.Task{}
			err := yaml.Unmarshal([]byte(tt.taskYaml), task)
			if err != nil {
				t.Errorf("Failed to unmarshal task yaml %s: %v", tt.taskYaml, err)
			}

			got, err := Build(task)
			if (err != nil) != tt.wantErr {
				t.Errorf("Build(%s) should've failed but hasn't: error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Build(%s) got = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
