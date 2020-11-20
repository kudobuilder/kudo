package podexec

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	// ErrCommandFailed is returned for command executions with an exit code > 0
	ErrCommandFailed = errors.New("command failed: ")
)

// PodExec defines a command that will be executed in a running Pod.
// RestCfg - The REST configuration for the cluster.
// PodName - The pod name on which to execute the command.
// PodNamespace - Namespace of the pod
// ContainerName - optional container to execute the command in. If empty, first container is taken
// Args - The command (and args) to execute.
// In - Command input stream.
// Out - Command output stream
// Err - Command error stream
type PodExec struct {
	RestCfg       *rest.Config
	PodName       string
	PodNamespace  string
	ContainerName string
	Args          []string
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
	TTY           bool
}

// Run executes a command in a pod. This is a distilled version of what `kubectl exec` (and
// also `kubectl  cp`) doing under the hood: a POST request is made to the `exec` subresource
// of the v1/pods endpoint containing the pod information and the command. Here is a good article
// describing it in detail: https://erkanerol.github.io/post/how-kubectl-exec-works/
//
// The result of the command execution is returned via passed PodExec.Out and PodExec.Err streams.
// Run calls the remotecommand executor, which executes the command in the remote pod, captures
// stdout and stderr, writes them to the provided Out and Err writers and then returns with an exit code.
// Note that when using SYNCHRONOUS io.Pipe for Out or Err streams Run call will not return until the
// streams are consumed. Here, Run has to be executed in a goroutine.
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
		TTY:       pe.TTY,
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
		Tty:    pe.TTY,
	}

	// The result of the executor.Stream() call itself is returned through the streams (In, Out and Err)
	// defined in the PodExec, e.g. when downloading a file, pe.Out will  contain the file bytes.
	return exec.Stream(so)
}

// FileSize fetches the size of a file in a remote pod. It runs `stat -c %s file` command in the
// pod and parses the output.
func FileSize(file string, pod *v1.Pod, ctrName string, restCfg *rest.Config) (int64, error) {
	stdout := strings.Builder{}

	pe := PodExec{
		RestCfg:       restCfg,
		PodName:       pod.Name,
		PodNamespace:  pod.Namespace,
		ContainerName: ctrName,
		Args:          []string{"stat", "-c", "%s", file},
		In:            nil,
		Out:           &stdout,
		Err:           nil,
		TTY:           true, // this will forward 2>&1. otherwise, reading from Out will never return for e.g. missing files
	}

	if err := pe.Run(); err != nil {
		return 0, fmt.Errorf("%wfailed to get the size of %s, err: %v, stderr: %s", ErrCommandFailed, file, err, stdout.String())
	}

	raw := stdout.String()
	trimmed := raw[:len(raw)-2] // remove trailing \n\r
	size, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the size '%s' of the file %s : %v", raw, file, err)
	}

	return size, nil
}

// DownloadFile fetches a file from a remote pod. It runs `tar cf - file` command and streams contents
// of the file via the stdout. Locally, the tar file is extracted into the passed afero filesystem where
// it is saved under the same path. Afero filesystem is used to allow the caller downloading and persisting
// of multiple files concurrently (afero filesystem is thread-safe).
func DownloadFile(fs afero.Fs, file string, pod *v1.Pod, ctrName string, restCfg *rest.Config) error {
	stdout := bytes.Buffer{}
	stderr := strings.Builder{}

	pe := PodExec{
		RestCfg:       restCfg,
		PodName:       pod.Name,
		PodNamespace:  pod.Namespace,
		ContainerName: ctrName,
		Args:          []string{"tar", "cf", "-", file},
		In:            nil,
		Out:           &stdout,
		Err:           &stderr,
	}
	if err := pe.Run(); err != nil {
		return fmt.Errorf("%wfailed to copy pipe file. err: %v, stderr: %s", ErrCommandFailed, err, stderr.String())
	}

	if err := untarFile(fs, &stdout, file); err != nil {
		return fmt.Errorf("failed to untar pipe file: %v", err)
	}

	return nil
}

// untarFile extracts a tar file from the passed reader using passed file name.
func untarFile(fs afero.Fs, r io.Reader, fileName string) error {
	tr := tar.NewReader(r)

	// Don't untar more than 4GiB to mitigate uncompression bombs.
	const writtenLimit int64 = 4294967000

	var written int64

	for {
		header, err := tr.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		// the target location of the file. tar strips the leading "/" however, we treat the pipe file path
		// as a key to the underlying data (otherwise we'll have to start splitting paths). To avoid all
		// the complexity and because we only extract one file here, the path is taken from the PipeFile configuration
		target := fileName

		// check the file type
		switch header.Typeflag {
		// if it's a file create it
		case tar.TypeReg:
			w, err := copyFile(tr, fs, target, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			written += w
			if written > writtenLimit {
				return errors.New("untar aborted because archive exceeds 4GiB")
			}

		default:
			log.Printf("skipping %s because it is not a regular file or a directory", header.Name)
		}
	}

	return nil
}

func copyFile(reader io.Reader, fs afero.Fs, target string, fileMode os.FileMode) (written int64, err error) {
	f, err := fs.OpenFile(target, os.O_CREATE|os.O_WRONLY, fileMode)
	if err != nil {
		return 0, err
	}

	defer func() {
		if ferr := f.Close(); ferr != nil {
			err = ferr
		}
	}()

	written, err = io.Copy(f, reader)

	return written, err
}

// HasCommandFailed returns true if PodExec command returned an exit code > 0
func HasCommandFailed(err error) bool {
	return err != nil && errors.Is(err, ErrCommandFailed)
}
