package cmd

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testcore "k8s.io/client-go/testing"
)

func TestInitCmd_dry(t *testing.T) {

	var buf bytes.Buffer

	cmd := &initCmd{
		out:    &buf,
		fs:     afero.NewMemMapFs(),
		dryRun: true,
	}
	if err := cmd.run(); err != nil {
		t.Errorf("expected error: %v", err)
	}
	expected := ""
	if !strings.Contains(buf.String(), expected) {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestInitCmd_exists(t *testing.T) {

	var buf bytes.Buffer
	fc := fake.NewSimpleClientset(&v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kudo-system",
			Name:      "kudo-manager-deploy",
		},
	})
	fc.PrependReactor("*", "*", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewAlreadyExists(v1.Resource("deployments"), "1")
	})
	cmd := &initCmd{
		out:    &buf,
		fs:     afero.NewMemMapFs(),
		client: &kube.Client{KubeClient: fc},
	}
	if err := cmd.run(); err != nil {
		t.Errorf("expected error: %v", err)
	}
	expected := "Warning: KUDO manager is already installed in the cluster.\n" +
		"(Use --client-only to suppress this message)"

	if !strings.Contains(buf.String(), expected) {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestInitCmd_output tests that init -o can be decoded
func TestInitCmd_output(t *testing.T) {

	fc := fake.NewSimpleClientset()
	tests := []string{"yaml"}
	for _, s := range tests {
		var buf bytes.Buffer
		cmd := &initCmd{
			out:    &buf,
			client: &kube.Client{KubeClient: fc},
			output: s,
			dryRun: true,
		}
		// ensure that we can marshal
		if err := cmd.run(); err != nil {
			t.Fatal(err)
		}
		// ensure no calls against the server
		if got := len(fc.Actions()); got != 0 {
			t.Errorf("expected no server calls, got %d", got)
		}

		assert.True(t, len(buf.Bytes()) > 0, "Buffer needs to have an output")
		// ensure we can decode what was created
		var obj interface{}
		decoder := yamlutil.NewYAMLOrJSONDecoder(&buf, 4096)
		for {
			err := decoder.Decode(&obj)
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("error decoding init %s output %s %s", s, err, buf.String())
			}
		}
	}
}

func Test_initCmd_YAMLWriter(t *testing.T) {
	type fields struct {
		out        io.Writer
		fs         afero.Fs
		image      string
		dryRun     bool
		output     string
		version    string
		wait       bool
		timeout    int64
		kubeClient kubernetes.Interface
	}
	type args struct {
		manifests []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &initCmd{
				out:     tt.fields.out,
				fs:      tt.fields.fs,
				image:   tt.fields.image,
				dryRun:  tt.fields.dryRun,
				output:  tt.fields.output,
				version: tt.fields.version,
				wait:    tt.fields.wait,
				timeout: tt.fields.timeout,
				client:  &kube.Client{KubeClient: tt.fields.kubeClient},
			}
			w := &bytes.Buffer{}
			err := i.YAMLWriter(w, tt.args.manifests)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("YAMLWriter() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func Test_initCmd_run(t *testing.T) {
	type fields struct {
		out        io.Writer
		fs         afero.Fs
		image      string
		dryRun     bool
		output     string
		version    string
		wait       bool
		timeout    int64
		kubeClient kubernetes.Interface
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &initCmd{
				out:     tt.fields.out,
				fs:      tt.fields.fs,
				image:   tt.fields.image,
				dryRun:  tt.fields.dryRun,
				output:  tt.fields.output,
				version: tt.fields.version,
				wait:    tt.fields.wait,
				timeout: tt.fields.timeout,
				client:  &kube.Client{KubeClient: tt.fields.kubeClient},
			}
			if err := i.run(); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newInitCmd(t *testing.T) {
	type args struct {
		fs afero.Fs
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
		want    *cobra.Command
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			got := newInitCmd(tt.args.fs, out)
			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("newInitCmd() gotOut = %v, want %v", gotOut, tt.wantOut)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newInitCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}
