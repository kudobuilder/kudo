package task

import (
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// PodExec defines a command that will be executed in a running Pod.
// RestCfg - The REST configuration for the cluster.
// PodName - The pod name on which to execute the command.
// PodNamespace - Namespace of the pod
// ContainerName - optional container to execute the command in. If empty, first container is taken
// Args - The command (and args) to execute.
// In - An (optional) command input stream.
// Out - The command output stream set by `Run()`.
// Err - the command error stream set by `Run()`.
type PodExec struct {
	RestCfg       *rest.Config
	PodName       string
	PodNamespace  string
	ContainerName string
	Args          []string
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
}

// Run executes a command in a pod. This is a distilled version of what `kubectl exec` (and
// also `kubectl  cp`) doing under the hood: a POST request is made to the `exec` subresource
// of the v1/pods endpoint containing the pod information and the command. Here is a good article
// describing it in detail: https://erkanerol.github.io/post/how-kubectl-exec-works/
func (pe *PodExec) Run() error {
	codec := serializer.NewCodecFactory(scheme.Scheme)
	restClient, err := apiutil.RESTClientForGVK(
		schema.GroupVersionKind{
			Version: "v1",
			Kind:    "pods",
		},
		pe.RestCfg,
		codec)
	if err != nil {
		return err
	}

	req := restClient.
		Post().
		Resource("pods").
		Name(pe.PodName).
		Namespace(pe.PodNamespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Stdin:     pe.In != nil,
		Stdout:    pe.Out != nil,
		Stderr:    pe.Err != nil,
		TTY:       false,
		Container: pe.ContainerName,
		Command:   pe.Args,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(pe.RestCfg, "POST", req.URL())
	if err != nil {
		return err
	}

	so := remotecommand.StreamOptions{
		Stdin:  pe.In,
		Stdout: pe.Out,
		Stderr: pe.Err,
		Tty:    false,
	}

	// Executor.Stream() call has to be made in a goroutine, otherwise it blocks the execution.
	// We don't wait for the execution to end: the result of the command is returned though the
	// streams (In, Out and Err) defined in the PodExec, e.g. when downloading a file, Out will
	// contain the file bytes.
	go func(exec remotecommand.Executor, so remotecommand.StreamOptions) {
		// TODO: we need to propagate this error to the caller
		err = exec.Stream(so)
		if err != nil {
			fmt.Printf("error during pod command %+v execution: %v", pe, err)
		}
	}(exec, so)

	return nil
}
