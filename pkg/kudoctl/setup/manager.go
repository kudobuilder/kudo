package setup

import (
	"strconv"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

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
)

//Defines the deployment of the KUDO manager and it's service definition.
type KudoManager struct {
	options    Options
	service    *v1.Service
	deployment *appsv1.StatefulSet
}

// Manager returns the setup management object
func Manager(options Options) KudoManager {
	return KudoManager{
		service:    generateService(options),
		deployment: generateDeployment(options),
	}
}

// Install uses Kubernetes client to install KUDO.
func (m KudoManager) Install(client *kube.Client) error {
	if err := m.installStatefulSet(client.KubeClient.AppsV1()); err != nil {
		return err
	}

	if err := m.installService(client.KubeClient.CoreV1()); err != nil {
		return err
	}
	return nil
}

func (m KudoManager) installStatefulSet(client appsv1client.StatefulSetsGetter) error {
	_, err := client.StatefulSets(m.options.Namespace).Create(m.deployment)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("statefulset %v already exists", m.deployment.Name)
		return nil
	}
	return err
}

func (m KudoManager) installService(client corev1.ServicesGetter) error {
	_, err := client.Services(m.options.Namespace).Create(m.service)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service %v already exists", m.service.Name)
		return nil
	}
	return err
}

// AsYamlManifests provides a slice of strings for the deployment and service manifest
func (m KudoManager) AsYamlManifests() ([]string, error) {
	s := m.service
	d := m.deployment

	objs := []runtime.Object{s, d}

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

// ManagerLabels returns the labels used by deployment and service
func ManagerLabels() labels.Set {
	return generateLabels(map[string]string{"control-plane": "controller-manager", "controller-tools.k8s.io": "1.0"})
}

func generateDeployment(opts Options) *appsv1.StatefulSet {
	managerLabels := ManagerLabels()

	secretDefaultMode := int32(420)
	image := opts.Image
	d := &appsv1.StatefulSet{
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
								{Name: "ENABLE_WEBHOOKS", Value: strconv.FormatBool(opts.hasWebhooksEnabled())},
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
							VolumeMounts: []v1.VolumeMount{
								{Name: "cert", MountPath: "/tmp/cert", ReadOnly: true},
							},
						},
					},
					TerminationGracePeriodSeconds: &opts.TerminationGracePeriodSeconds,
					Volumes: []v1.Volume{
						{
							Name: "cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName:  "kudo-webhook-server-secret",
									DefaultMode: &secretDefaultMode,
								},
							},
						},
					},
				},
			},
		},
	}

	return d
}

func generateService(opts Options) *v1.Service {
	managerLabels := ManagerLabels()
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
