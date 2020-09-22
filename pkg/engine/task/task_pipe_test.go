package task

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
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
		test := test

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
		tt := tt

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
			name: "a valid pipe-pod with one init container",
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
			name: "an invalid pipe-pod with a container",
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
			name: "an invalid pipe-pod with wrong volume mount",
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
			name: "a valid pipe-pod with two volumes, one of which is emptyDir type",
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
			name: "an invalid pipe-pod with two emptyDir volumes",
			podYaml: `
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: shared-data-one
    emptyDir: {}
  - name: shared-data-two
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - touch /tmp/foo.txt
      volumeMounts:
        - name: shared-data-one
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
			name: "an invalid pipe-pod where init container does not mount shared volume",
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
			name: "an invalid pipe-pod where init container does not mount shared volume",
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			pod, err := unmarshal(tt.podYaml)
			assert.NoError(t, err, "error during pipe pod unmarshaling")

			if err := validate(pod, tt.ff); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_pipeFiles(t *testing.T) {
	meta := renderer.Metadata{
		Metadata:  engine.Metadata{InstanceName: "foo-instance"},
		PlanName:  "deploy",
		PhaseName: "first",
		StepName:  "step",
		TaskName:  "genfiles",
	}

	tests := []struct {
		name         string
		data         map[string]string
		file         PipeFile
		meta         renderer.Metadata
		wantArtifact interface{}
		wantErr      bool
	}{
		{
			name: "pipe a file to a secret",
			data: map[string]string{"/tmp/foo.txt": "foo"},
			file: PipeFile{
				File: "/tmp/foo.txt",
				Kind: PipeFileKindSecret,
				Key:  "Foo",
			},
			meta: meta,
			wantArtifact: v1.Secret{
				TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "fooinstance.deploy.first.step.genfiles.foo"},
				Data:       map[string][]byte{"foo.txt": []byte("foo")},
				Type:       v1.SecretTypeOpaque,
			},
			wantErr: false,
		},
		{
			name: "pipe an env file to a secret",
			data: map[string]string{"/tmp/foo.env": `
				enemies=aliens
				lives=3
				allowed="true"
			`},
			file: PipeFile{
				EnvFile: "/tmp/foo.env",
				Kind:    PipeFileKindSecret,
				Key:     "Foo",
			},
			meta: meta,
			wantArtifact: v1.Secret{
				TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "fooinstance.deploy.first.step.genfiles.foo"},
				Data: map[string][]byte{
					"enemies": []byte("aliens"),
					"lives":   []byte("3"),
					"allowed": []byte("\"true\""),
				},
				Type: v1.SecretTypeOpaque,
			},
			wantErr: false,
		},
		{
			name: "pipe a file to a configMap",
			data: map[string]string{"/tmp/bar.txt": "bar"},
			file: PipeFile{
				File: "/tmp/bar.txt",
				Kind: PipeFileKindConfigMap,
				Key:  "Bar",
			},
			meta: meta,
			wantArtifact: v1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "fooinstance.deploy.first.step.genfiles.bar"},
				BinaryData: map[string][]byte{"bar.txt": []byte("bar")},
			},
			wantErr: false,
		},
		{
			name: "pipe an env file to a configMap",
			data: map[string]string{"/tmp/bar.env": `
				enemies=aliens
				lives=3
				allowed="true"
			`},
			file: PipeFile{
				EnvFile: "/tmp/bar.env",
				Kind:    PipeFileKindConfigMap,
				Key:     "Bar",
			},
			meta: meta,
			wantArtifact: v1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "fooinstance.deploy.first.step.genfiles.bar"},
				Data: map[string]string{
					"enemies": "aliens",
					"lives":   "3",
					"allowed": "\"true\"",
				},
			},
			wantErr: false,
		},
		{
			name: "return an error for an invalid pipe",
			data: map[string]string{"nope.txt": ""},
			file: PipeFile{
				File: "nope.txt",
				Kind: "Invalid",
				Key:  "Nope",
			},
			meta:         meta,
			wantArtifact: nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			for path, data := range tt.data {
				assert.NoError(t, afero.WriteFile(fs, path, []byte(data), 0644), "error while preparing test: %s", tt.name)
			}

			got, err := createArtifacts(fs, []PipeFile{tt.file}, tt.meta)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("createArtifacts() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			out, err := yaml.Marshal(tt.wantArtifact)
			assert.NoError(t, err, "failure during marshaling of the test pipe artifact in test: %s", tt.name)

			want := map[string]string{tt.file.Key: string(out)}
			assert.Equal(t, want, got, "createArtifacts() unexpected return value")
		})
	}
}
