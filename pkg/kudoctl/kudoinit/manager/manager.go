package manager

import (
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verify"
)

// Ensure kudoinit.Step is implemented
var _ kudoinit.Step = &Initializer{}

//Defines the deployment of the KUDO manager and it's service definition.
type Initializer struct {
	options    kudoinit.Options
	service    *v1.Service
	deployment *appsv1.StatefulSet
}

// NewInitializer returns the setup management object
func NewInitializer(options kudoinit.Options) Initializer {
	return Initializer{
		options:    options,
		service:    generateService(options),
		deployment: generateDeployment(options),
	}
}

func (m Initializer) PreInstallVerify(client *kube.Client) verify.Result {
	return verify.NewResult()
}

func (m Initializer) String() string {
	return "kudo controller"
}

// Install uses Kubernetes client to install KUDO.
func (m Initializer) Install(client *kube.Client) error {
	if err := m.installStatefulSet(client.KubeClient.AppsV1()); err != nil {
		return err
	}

	if err := m.installService(client.KubeClient.CoreV1()); err != nil {
		return err
	}
	return nil
}

func (m Initializer) installStatefulSet(client appsv1client.StatefulSetsGetter) error {
	_, err := client.StatefulSets(m.options.Namespace).Create(m.deployment)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("statefulset %v already exists", m.deployment.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("stateful set: %v", err)
	}
	return err
}

func (m Initializer) installService(client corev1.ServicesGetter) error {
	_, err := client.Services(m.options.Namespace).Create(m.service)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service %v already exists", m.service.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("service: %v", err)
	}
	return err
}

func (m Initializer) AsArray() []runtime.Object {
	return []runtime.Object{m.service, m.deployment}
}

// AsYamlManifests provides a slice of strings for the deployment and service manifest
func (m Initializer) AsYamlManifests() ([]string, error) {
	objs := m.AsArray()

	manifests := make([]string, len(objs))
	for i, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}

// GenerateLabels returns the labels used by deployment and service
func GenerateLabels() labels.Set {
	return kudoinit.GenerateLabels(map[string]string{"control-plane": "controller-manager"})
}

func generateDeployment(opts kudoinit.Options) *appsv1.StatefulSet {
	managerLabels := GenerateLabels()

	secretDefaultMode := int32(420)
	image := opts.Image
	s := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      "kudo-controller-manager",
			Labels:    managerLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector:    &metav1.LabelSelector{MatchLabels: managerLabels},
			ServiceName: "kudo-controller-manager-service",
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: managerLabels,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: opts.ServiceAccount,
					Containers: []v1.Container{
						{
							Command: []string{"/root/manager"},
							Env: []v1.EnvVar{
								{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "SECRET_NAME", Value: "kudo-webhook-server-secret"},
								{Name: "ENABLE_WEBHOOKS", Value: strconv.FormatBool(opts.HasWebhooksEnabled())},
							},
							Image:           image,
							ImagePullPolicy: "Always",
							Name:            "manager",
							Ports: []v1.ContainerPort{
								// name matters for service
								{ContainerPort: 443, Name: "webhook-server", Protocol: "TCP"},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("50Mi")},
							},
						},
					},
					TerminationGracePeriodSeconds: &opts.TerminationGracePeriodSeconds,
				},
			},
		},
	}

	if opts.HasWebhooksEnabled() {
		s.Spec.Template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{Name: "cert", MountPath: "/tmp/cert", ReadOnly: true},
		}
		s.Spec.Template.Spec.Volumes = []v1.Volume{
			{
				Name: "cert",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName:  "kudo-webhook-server-secret",
						DefaultMode: &secretDefaultMode,
					},
				},
			},
		}
	}

	return s
}

func generateService(opts kudoinit.Options) *v1.Service {
	managerLabels := GenerateLabels()
	s := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      "kudo-controller-manager-service",
			Labels:    managerLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "kudo",
					Port:       443,
					TargetPort: intstr.FromString("webhook-server")},
			},
			Selector: managerLabels,
		},
	}
	return s
}
