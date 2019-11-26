package task

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var (
	pipeTaskError = "PipeTaskError"

	pipePodContainerName = "waiter"
)

type PipeTask struct {
	Name      string
	Container string
	PipeFiles []PipeFile
}

type PipeFile struct {
	File string
	Kind string
	Key  string
}

func (pt PipeTask) Run(ctx Context) (bool, error) {
	// 1. - Render container template -
	rendered, err := render([]string{pt.Container}, ctx.Templates, ctx.Parameters, ctx.Meta)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Create core/v1 container object -
	container, err := unmarshal(rendered[pt.Container])
	if err != nil {
		return false, fatalExecutionError(err, resourceUnmarshalError, ctx.Meta)
	}

	// 3. - Validate the container object -
	err = validate(container, pt.PipeFiles)
	if err != nil {
		return false, fatalExecutionError(err, resourceValidationError, ctx.Meta)
	}

	// 4. - Create a pod using the container -
	podName := podName(ctx)
	podStr, err := pipePod(container, podName)
	if err != nil {
		return false, fatalExecutionError(err, pipeTaskError, ctx.Meta)
	}

	// 5. - Kustomize pod with metadata
	podObj, err := kustomize(map[string]string{"pipe-pod.yaml": podStr}, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, fatalExecutionError(err, taskEnhancementError, ctx.Meta)
	}

	// 6. - Apply pod using the client -
	podObj, err = apply(podObj, ctx.Client)
	if err != nil {
		return false, err
	}

	// 7. - Wait for the pod to be ready -
	err = isHealthy(podObj)
	// once the pod is Ready, it means that its initContainer finished successfully and we can copy
	// out the generated files. An error during a health check is not treated as task execution error
	if err != nil {
		return false, nil
	}

	// 8. - Copy out the pipe files -
	log.Printf("PipeTask: %s/%s copying pipe files", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	fs := afero.NewMemMapFs()
	pipePod := podObj[0].(*corev1.Pod)

	err = copyFiles(fs, pt.PipeFiles, pipePod, ctx)
	if err != nil {
		return false, err
	}

	// 9. - Create k8s artifacts (ConfigMap/Secret) from the pipe files -
	log.Printf("PipeTask: %s/%s creating pipe artifacts", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	artStr, err := pipeFiles(fs, pt.PipeFiles, ctx)
	if err != nil {
		return false, err
	}

	// 10. - Kustomize artifacts -
	artObj, err := kustomize(artStr, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, fatalExecutionError(err, taskEnhancementError, ctx.Meta)
	}

	// 11. - Apply artifacts using the client -
	_, err = apply(artObj, ctx.Client)
	if err != nil {
		return false, err
	}

	// 12. - Delete pipe pod -
	log.Printf("PipeTask: %s/%s deleting pipe pod", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	err = delete(podObj, ctx.Client)
	if err != nil {
		return false, err
	}

	return true, nil
}

func unmarshal(ctrStr string) (*corev1.Container, error) {
	ctr := &corev1.Container{}
	err := yaml.Unmarshal([]byte(ctrStr), ctr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall pipe container: %v", err)
	}
	return ctr, nil
}

var pipeFileRe = regexp.MustCompile(`[-._a-zA-Z0-9]+`)

func isRelative(base, file string) bool {
	rp, err := filepath.Rel(base, file)
	return err == nil && !strings.HasPrefix(rp, ".")
}

// validate method validates passed pipe container. It is expected to:
// - have exactly one volume mount
// - not have readiness probes (as k87 does not support them for init containers)
// - pipe files should have valid names and exist within the volume mount
func validate(ctr *corev1.Container, ff []PipeFile) error {
	if len(ctr.VolumeMounts) != 1 {
		return errors.New("pipe container should have exactly one volume mount")
	}

	if ctr.ReadinessProbe != nil {
		return errors.New("pipe container does not support readiness probes")
	}

	// check if all referenced pipe files are children of the container mountPath
	mountPath := ctr.VolumeMounts[0].MountPath
	for _, f := range ff {
		if !isRelative(mountPath, f.File) {
			return fmt.Errorf("pipe file %s should be a child of %s mount path", f.File, mountPath)
		}

		fileName := path.Base(f.File)
		// Same as K87 we use file names as ConfigMap data keys. A valid key name for a ConfigMap must consist
		// of alphanumeric characters, '-', '_' or '.' (e.g. 'key.name',  or 'KEY_NAME',  or 'key-name', regex
		// used for validation is '[-._a-zA-Z0-9]+')
		if !pipeFileRe.MatchString(fileName) {
			return fmt.Errorf("pipe file name %s should only contain alphanumeric characters, '.', '_' and '-'", fileName)
		}
	}
	return nil
}

func pipePod(ctr *corev1.Container, name string) (string, error) {
	volumeName := ctr.VolumeMounts[0].Name
	mountPath := ctr.VolumeMounts[0].MountPath

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name:         volumeName,
					VolumeSource: corev1.VolumeSource{EmptyDir: nil},
				},
			},
			InitContainers: []corev1.Container{*ctr},
			Containers: []corev1.Container{
				{
					Name:    pipePodContainerName,
					Image:   "busybox",
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{"sleep infinity"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: mountPath,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}

	b, err := yaml.Marshal(pod)
	if err != nil {
		return "", fmt.Errorf("failed to create pipe task pod: %v", err)
	}

	return string(b), nil
}

func copyFiles(fs afero.Fs, ff []PipeFile, pod *corev1.Pod, ctx Context) error {
	restCfg, err := config.GetConfig()
	if err != nil {
		return fatalExecutionError(fmt.Errorf("failed to fetch cluster REST config: %v", err), pipeTaskError, ctx.Meta)
	}

	var g errgroup.Group

	for _, f := range ff {
		f := f
		log.Printf("PipeTask: %s/%s copying pipe file %s", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName, f.File)
		g.Go(func() error {
			return copyFile(fs, f, pod, restCfg)
		})
	}

	err = g.Wait()
	return err
}

func copyFile(fs afero.Fs, pf PipeFile, pod *corev1.Pod, restCfg *rest.Config) error {
	reader, stdout := io.Pipe()

	defer reader.Close()
	defer stdout.Close()

	pe := PodExec{
		RestCfg:       restCfg,
		PodName:       pod.Name,
		PodNamespace:  pod.Namespace,
		ContainerName: pipePodContainerName,
		Args:          []string{"tar", "cf", "-", pf.File},
		In:            nil,
		Out:           stdout,
		Err:           nil,
	}

	if err := pe.Run(); err != nil {
		return fmt.Errorf("failed to copy pipe file. err: %v", err)
	}

	if err := untarFile(fs, reader, pf.File); err != nil {
		return fmt.Errorf("failed to untar pipe file: %v", err)
	}

	return nil
}

// pipeFiles iterates through passed pipe files and their copied data, reads them, constructs k8s artifacts
// and marshals them.
func pipeFiles(fs afero.Fs, files []PipeFile, ctx Context) (map[string]string, error) {
	artifacts := map[string]string{}

	for _, pf := range files {
		data, err := afero.ReadFile(fs, pf.File)
		if err != nil {
			return nil, fmt.Errorf("error opening pipe file %s", pf.File)
		}

		// API server has a limit of 1Mb for Secret/ConfigMap
		if len(data) > 1024*1024 {
			return nil, fatalExecutionError(fmt.Errorf("pipe file %s size (%d bytes) exceeds max size limit of 1Mb", pf.File, len(data)), pipeTaskError, ctx.Meta)
		}

		var art string
		switch pf.Kind {
		case "Secret":
			art, err = pipeSecret(pf, data, ctx)
		case "ConfigMap":
			art, err = pipeConfigMap(pf, data, ctx)
		default:
			return nil, fmt.Errorf("unknown pipe file kind: %+v", pf)
		}

		if err != nil {
			return nil, err
		}
		artifacts[pf.Key] = art
	}

	return artifacts, nil
}

// pipeSecret method creates a core/v1/Secret object using passed data. Pipe file name is used
// as Secret data key. Secret name will be of the form <instance>.<plan>.<phase>.<step>.<task>-<PipeFile.Key>
func pipeSecret(pf PipeFile, data []byte, ctx Context) (string, error) {
	name := artifactName(ctx, pf.Key)
	key := path.Base(pf.File)

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{key: data},
		Type: corev1.SecretTypeOpaque,
	}

	b, err := yaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipe secret for pipe file %s: %v", pf.File, err)
	}

	return string(b), nil
}

// pipeConfigMap method creates a core/v1/ConfigMap object using passed data. Pipe file name is used
// as ConfigMap data key. ConfigMap name will be of the form <instance>.<plan>.<phase>.<step>.<task>-<PipeFile.Key>
func pipeConfigMap(pf PipeFile, data []byte, ctx Context) (string, error) {
	name := artifactName(ctx, pf.Key)
	key := path.Base(pf.File)

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		BinaryData: map[string][]byte{key: data},
	}

	b, err := yaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipe configMap for pipe file %s: %v", pf.File, err)
	}

	return string(b), nil
}

// untarFile extracts a tar file from the passed reader using passed file name.
func untarFile(fs afero.Fs, r io.Reader, fileName string) (err error) {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF: // if no more files are found return
			return nil

		case err != nil: // return any other error
			return err

		case header == nil: // if the header is nil, just skip it
			continue
		}

		// the target location of the file. tar strips the leading "/" however, we treat the pipe file path
		// as a key to the underlying data (otherwise we'll have to start splitting paths). To avoid all
		// the complexity and because we only extract one file here, the path is taken from the PipeFile configuration
		target := fileName

		// check the file type
		switch header.Typeflag {
		// if it's a file create it
		case tar.TypeReg:
			f, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; deferring would cause each file close
			// to wait until all operations have completed.
			f.Close() // nolint

		default:
			fmt.Printf("skipping %s because it is not a regular file or a directory", header.Name)
		}
	}
}

// podName returns a deterministic name for a pipe pod
func podName(ctx Context) string { return name(ctx, "pipe-pod") }

// artifactName returns a deterministic name for pipe artifact (ConfigMap, Secret)
func artifactName(ctx Context, key string) string { return name(ctx, key) }

var (
	nameRe = regexp.MustCompile(`^[^a-zA-Z0-9\-.]+`)
	// TODO: this is the reqexp that API server is using for Secret/ConfigMap name validation
	//artifactNameRe = regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)
)

// name returns a deterministic names for pipe artifacts (Pod, Secret, ConfigMap) in the form:
// <instance>.<plan>.<phase>.<step>.<task>-<suffix> All characters aside from these defined in
// nameRe are removed and replaced with ""
func name(ctx Context, suffix string) string {
	name := fmt.Sprintf("%s.%s.%s.%s.%s-%s",
		ctx.Meta.InstanceName,
		ctx.Meta.PlanName,
		ctx.Meta.PhaseName,
		ctx.Meta.StepName,
		ctx.Meta.TaskName,
		suffix)
	return nameRe.ReplaceAllString(strings.ToLower(name), "")
}
