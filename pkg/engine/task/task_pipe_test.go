package task

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/stretchr/testify/assert"
)

func Test_isRelative(t *testing.T) {
	tests := []struct {
		base     string
		file     string
		relative bool
	}{
		{
			base:     "/dir",
			file:     "/dir/../link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/dir/../../link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/link",
			relative: false,
		},
		{
			base:     "/dir",
			file:     "/dir/link",
			relative: true,
		},
		{
			base:     "/dir",
			file:     "/dir/int/../link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/int/../link",
			relative: true,
		},
		{
			base:     "dir",
			file:     "dir/../../link",
			relative: false,
		},
		{
			base:     "/tmp",
			file:     "/tmp/foo.txt",
			relative: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if test.relative != isRelative(test.base, test.file) {
				t.Errorf("unexpected result for: base %q, file %q", test.base, test.file)
			}
		})
	}
}

func TestPipeNames(t *testing.T) {
	tests := []struct {
		name             string
		meta             renderer.Metadata
		key              string
		wantPodName      string
		wantArtifactName string
	}{
		{
			name: "simple",
			meta: renderer.Metadata{
				Metadata:  engine.Metadata{InstanceName: "foo-instance"},
				PlanName:  "deploy",
				PhaseName: "first",
				StepName:  "step",
				TaskName:  "genfiles",
			},
			key:              "foo",
			wantPodName:      "fooinstance.deploy.first.step.genfiles.pipepod",
			wantArtifactName: "fooinstance.deploy.first.step.genfiles.foo",
		},
		{
			name: "with invalid characters",
			meta: renderer.Metadata{
				Metadata:  engine.Metadata{InstanceName: "Foo-Instance"},
				PlanName:  "deploy",
				PhaseName: "first",
				StepName:  "step",
				TaskName:  "gen_files",
			},
			key:              "$!Foo%",
			wantPodName:      "fooinstance.deploy.first.step.genfiles.pipepod",
			wantArtifactName: "fooinstance.deploy.first.step.genfiles.foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PipePodName(tt.meta); got != tt.wantPodName {
				t.Errorf("PipePodName() = %v, want %v", got, tt.wantPodName)
			}

			if got := PipeArtifactName(tt.meta, tt.key); got != tt.wantArtifactName {
				t.Errorf("PipeArtifactName() = %v, want %v", got, tt.wantArtifactName)
			}
		})
	}
}

func Test_validate(t *testing.T) {
	tests := []struct {
		name    string
		podYaml string
		ff      []PipeFile
		wantErr bool
	}{
		{
			name: "a valid pipe pod with one init container",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: shared-data
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/tmp/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: false,
		},
		{
			name: "an valid pipe pod with a container",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: shared-data
    emptyDir: {}

  containers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/tmp/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: true,
		},
		{
			name: "an invalid pipe pod with wrong volume mount",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: conf-data
    configMap:
      name: my-conf		

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/tmp/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: true,
		},
		{
			name: "a valid pipe pod with at least one emptyDir volume mount",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: conf-data
    configMap:
      name: my-conf
  - name: shared-data
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/tmp/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: false,
		},
		{
			name: "an invalid pipe pod where init container does not mount shared volume",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: conf-data
    configMap:
      name: my-conf
  - name: shared-data
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: conf-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/tmp/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: true,
		},
		{
			name: "an invalid pipe pod where init container does not mount shared volume",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: conf-data
    configMap:
      name: my-conf
  - name: shared-data
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
`,
			ff: []PipeFile{
				{
					File: "/var/foo.txt",
					Kind: PipeFileKindSecret,
					Key:  "foo",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod, err := unmarshal(tt.podYaml)
			assert.NoError(t, err, "error during pipe pod unmarshaling")

			if err := validate(pod, tt.ff); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
