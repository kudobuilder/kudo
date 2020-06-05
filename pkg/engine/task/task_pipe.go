package task

import (
	"errors"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/engine/task/podexec"
)

const (
	pipeTaskError = "PipeTaskError"

	// name of the main pipe pod container
	pipePodContainerName = "waiter"

	PipePodAnnotation = "kudo.dev/pipepod"
)

const (
	// Max file size of a pipe file.
	maxPipeFileSize = 1024 * 1024
)

type PipeFileKind string

const (
	// PipeFile will be persisted as a Secret
	PipeFileKindSecret PipeFileKind = "Secret"
	// PipeFile will be persisted as a ConfigMap
	PipeFileKindConfigMap PipeFileKind = "ConfigMap"
)

type PipeTask struct {
	Name      string
	Pod       string
	PipeFiles []PipeFile
}

type PipeFile struct {
	File    string
	EnvFile string
	Kind    PipeFileKind
	Key     string
}

// fileSource return either File or EnvFile depending on which one of them is set. Note that only one can be set at a
// time which is enforced by the validation.
func (pf PipeFile) fileSource() string {
	if pf.File != "" {
		return pf.File
	}
	return pf.EnvFile
}

func (pt PipeTask) Run(ctx Context) (bool, error) {
	// 1. - Render container template -
	rendered, err := render([]string{pt.Pod, pt.Pod}, ctx)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Create core/v1 container object -
	usrPod, err := unmarshal(rendered[pt.Pod])
	if err != nil {
		return false, fatalExecutionError(err, resourceUnmarshalError, ctx.Meta)
	}

	// 3. - Validate the container object -
	err = validate(usrPod, pt.PipeFiles)
	if err != nil {
		return false, fatalExecutionError(err, resourceValidationError, ctx.Meta)
	}

	// 4. - Create a pod using the container -
	podName := PipePodName(ctx.Meta)
	podYaml, err := pipePod(usrPod, podName)
	if err != nil {
		return false, fatalExecutionError(err, pipeTaskError, ctx.Meta)
	}

	// 5. - Convert to object
	objs, err := convert(map[string]string{"pipe-pod.yaml": podYaml})
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 6. - Enhance pod with metadata
	podObj, err := enhance(objs, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, err
	}

	// 7. - Apply pod using the client -
	podObj, err = applyResources(podObj, ctx)
	if err != nil {
		return false, err
	}

	// 8. - Wait for the pod to be ready -
	err = isHealthy(podObj)
	// once the pod is Ready, it means that its initContainer finished successfully and we can copy
	// out the generated files. An error during a health check is not treated as task execution error
	if err != nil {
		// our pod can not fail terminally, so we treat it as a transient error
		log.Printf("TaskExecution: %v", err)
		return false, nil
	}

	// 9. - Copy out the pipe files -
	log.Printf("PipeTask: %s/%s copying pipe files", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	fs := afero.NewMemMapFs()
	pipePod, ok := podObj[0].(*corev1.Pod)
	if !ok {
		return false, errors.New("internal error: pipe pod changed type after enhance and apply")
	}

	err = copyFiles(fs, pt.PipeFiles, pipePod, ctx)
	if err != nil {
		return false, err
	}

	// 10. - Create k8s artifacts (ConfigMap/Secret) from the pipe files -
	log.Printf("PipeTask: %s/%s creating pipe artifacts", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	artStr, err := createArtifacts(fs, pt.PipeFiles, ctx.Meta)
	if err != nil {
		return false, err
	}

	// 11. - Convert to objs
	artObjs, err := convert(artStr)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 12. - Enhance artifacts -
	artObj, err := enhance(artObjs, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, err
	}

	// 13. - Apply artifacts using the client -
	_, err = applyResources(artObj, ctx)
	if err != nil {
		return false, err
	}

	// 14. - Delete pipe pod -
	log.Printf("PipeTask: %s/%s deleting pipe pod", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName)
	err = deleteResource(podObj, ctx.Client)
	if err != nil {
		return false, err
	}

	return true, nil
}

func unmarshal(podStr string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := yaml.Unmarshal([]byte(podStr), pod)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall pipe pod: %v", err)
	}
	return pod, nil
}

var pipeFileRe = regexp.MustCompile(`[-._a-zA-Z0-9]+`)

func isRelative(base, file string) bool {
	rp, err := filepath.Rel(base, file)
	return err == nil && !strings.HasPrefix(rp, ".")
}

// sharedVolumeName method searches pod volumes for one of the type emptyDir and returns
// its name. Method expects exactly one such volume to exits and will return an error otherwise.
func sharedVolumeName(pod *corev1.Pod) (string, error) {
	volumes := []string{}
	for _, v := range pod.Spec.Volumes {
		if v.EmptyDir != nil {
			volumes = append(volumes, v.Name)
		}
	}
	if len(volumes) != 1 {
		return "", errors.New("pipe pod should define one emptyDir shared volume where the artifacts are temporary stored")
	}
	return volumes[0], nil
}

// sharedMountPath method searches pod initContainer volume mounts for one with a passed name.
// It returns the mount path of the volume if found or an error otherwise.
func sharedMountPath(pod *corev1.Pod, volName string) (string, error) {
	for _, vm := range pod.Spec.InitContainers[0].VolumeMounts {
		if vm.Name == volName {
			return vm.MountPath, nil
		}
	}

	return "", fmt.Errorf("pipe pod should save generated artifacts in %s", volName)
}

// validate method validates passed pipe pod. It is expected to:
// - have one init container and zero containers specified
// - one emptyDir shared volume should be defined where the artifacts will be stored
// - shared volume should be mounted in the init container
// - pipe files should have valid names and exist within mounted shared volume
// - pod should not have a RestartPolicy defined (or define an "OnFailure" one)
func validate(pod *corev1.Pod, ff []PipeFile) error {
	if len(pod.Spec.Containers) > 0 {
		return errors.New("pipe pod should not have containers. pipe artifacts are generated in the init container")
	}

	if len(pod.Spec.InitContainers) != 1 {
		return errors.New("pipe pod should define one init container that generated artifacts defined")
	}

	if pod.Spec.RestartPolicy != "" && pod.Spec.RestartPolicy != corev1.RestartPolicyOnFailure {
		return errors.New("pipe pod RestartPolicy should be OnFailure")
	}

	sharedVolName, err := sharedVolumeName(pod)
	if err != nil {
		return err
	}

	mountPath, err := sharedMountPath(pod, sharedVolName)
	if err != nil {
		return err
	}

	// check if all referenced pipe files are children of the container mountPath
	for _, f := range ff {
		if !isRelative(mountPath, f.fileSource()) {
			return fmt.Errorf("pipe file %s should be a child of %s mount path", f.fileSource(), mountPath)
		}

		fileName := path.Base(f.fileSource())
		// Same as k8s we use file names as ConfigMap data keys. A valid key name for a ConfigMap must consist
		// of alphanumeric characters, '-', '_' or '.' (e.g. 'key.name',  or 'KEY_NAME',  or 'key-name', regex
		// used for validation is '[-._a-zA-Z0-9]+')
		if !pipeFileRe.MatchString(fileName) {
			return fmt.Errorf("pipe file name %s should only contain alphanumeric characters, '.', '_' and '-'", fileName)
		}
	}
	return nil
}

func pipePod(pod *corev1.Pod, name string) (string, error) {
	volumeName, _ := sharedVolumeName(pod)
	mountPath, _ := sharedMountPath(pod, volumeName)

	if pod.GetAnnotations() == nil {
		pod.SetAnnotations(make(map[string]string))
	}
	pod.Annotations[PipePodAnnotation] = "true"
	pod.ObjectMeta.Name = name
	pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	pod.Spec.Containers = []corev1.Container{
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
	}

	b, err := yaml.Marshal(pod)
	if err != nil {
		return "", fmt.Errorf("failed to create pipe task pod: %v", err)
	}

	return string(b), nil
}

func copyFiles(fs afero.Fs, ff []PipeFile, pod *corev1.Pod, ctx Context) error {
	var g errgroup.Group

	for _, f := range ff {
		f := f
		log.Printf("PipeTask: %s/%s copying pipe file %s", ctx.Meta.InstanceNamespace, ctx.Meta.InstanceName, f.fileSource())
		g.Go(func() error {
			// Check the size of the pipe file first. K87 has a inherent limit on the size of
			// Secret/ConfigMap, so we avoid unnecessary copying of files that are too big by
			// checking its size first.
			size, err := podexec.FileSize(f.fileSource(), pod, pipePodContainerName, ctx.Config)
			if err != nil {
				// Any remote command exit code > 0 is treated as a fatal error since retrying it doesn't make sense
				if podexec.HasCommandFailed(err) {
					return fatalExecutionError(err, pipeTaskError, ctx.Meta)
				}
				return err
			}

			if size > maxPipeFileSize {
				return fatalExecutionError(fmt.Errorf("pipe file %s size %d exceeds maximum file size of %d bytes", f.fileSource(), size, maxPipeFileSize), pipeTaskError, ctx.Meta)
			}

			if err = podexec.DownloadFile(fs, f.fileSource(), pod, pipePodContainerName, ctx.Config); err != nil {
				// Any remote command exit code > 0 is treated as a fatal error since retrying it doesn't make sense
				if podexec.HasCommandFailed(err) {
					return fatalExecutionError(err, pipeTaskError, ctx.Meta)
				}
				return err
			}
			return nil
		})
	}

	return g.Wait()
}

// createArtifacts iterates through passed pipe files and their copied data, reads them, constructs k8s artifacts
// and marshals them.
func createArtifacts(fs afero.Fs, files []PipeFile, meta renderer.Metadata) (map[string]string, error) {
	artifacts := map[string]string{}

	for _, pf := range files {
		data, err := afero.ReadFile(fs, pf.fileSource())
		if err != nil {
			return nil, fmt.Errorf("error opening pipe file %s", pf.fileSource())
		}

		var art string
		switch pf.Kind {
		case PipeFileKindSecret:
			art, err = createSecret(pf, data, meta)
		case PipeFileKindConfigMap:
			art, err = createConfigMap(pf, data, meta)
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

// createSecret method creates a core/v1/Secret object using passed data. Pipe file name is used
// as Secret data key. Secret name will be of the form <instance>.<plan>.<phase>.<step>.<task>-<PipeFile.Key>
func createSecret(pf PipeFile, data []byte, meta renderer.Metadata) (string, error) {
	name := PipeArtifactName(meta, pf.Key)
	key := path.Base(pf.fileSource())

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	if pf.File != "" {
		secret.Data = map[string][]byte{key: data}
	}
	if pf.EnvFile != "" {
		err := addFromEnvFile(data, func(key, value string) {
			secret.Data[key] = []byte(value)
		})

		if err != nil {
			return "", fmt.Errorf("failed to read env var file %q: %v", pf.fileSource(), err)
		}
	}

	b, err := yaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipe secret for pipe file %s: %v", pf.File, err)
	}

	return string(b), nil
}

// createConfigMap method creates a core/v1/ConfigMap object using passed data. Pipe file name is used
// as ConfigMap data key. ConfigMap name will be of the form <instance>.<plan>.<phase>.<step>.<task>-<PipeFile.Key>
func createConfigMap(pf PipeFile, data []byte, meta renderer.Metadata) (string, error) {
	name := PipeArtifactName(meta, pf.Key)
	key := path.Base(pf.fileSource())

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data:       map[string]string{},
		BinaryData: map[string][]byte{},
	}

	if pf.File != "" {
		configMap.BinaryData = map[string][]byte{key: data}
	}
	if pf.EnvFile != "" {
		err := addFromEnvFile(data, func(key, value string) {
			configMap.Data[key] = value
		})

		if err != nil {
			return "", fmt.Errorf("failed to read env var file %q: %v", pf.fileSource(), err)
		}
	}

	b, err := yaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipe configMap for pipe file %s: %v", pf.File, err)
	}

	return string(b), nil
}

// PipePodName returns a deterministic name for a pipe pod
func PipePodName(meta renderer.Metadata) string { return name(meta, "pipepod") }

// PipeArtifactName returns a deterministic name for pipe artifact (ConfigMap, Secret)
func PipeArtifactName(meta renderer.Metadata, key string) string { return name(meta, key) }

var (
	alnum = regexp.MustCompile(`[^a-z0-9]+`)
)

// name returns a deterministic names for pipe artifacts (Pod, Secret, ConfigMap) in the form:
// <instance>.<plan>.<phase>.<step>.<task>.<suffix> All non alphanumeric characters are removed.
// A name for e.g a ConfigMap has to match a DNS-1123 subdomain, must consist of lower case alphanumeric
// characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com',
// regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
func name(meta renderer.Metadata, suffix string) string {
	sanitize := func(s string) string {
		return alnum.ReplaceAllString(strings.ToLower(s), "")
	}

	var parts []string
	for _, s := range []string{meta.InstanceName, meta.PlanName, meta.PhaseName, meta.StepName, meta.TaskName, suffix} {
		parts = append(parts, sanitize(s))
	}
	return strings.Join(parts, ".")
}
