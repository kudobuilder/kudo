package init

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/version"

	"k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	clientv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"
)

//Defines the deployment of the KUDO manager and it's service definition.

const (
	group              = "kudo.dev"
	crdVersion         = "v1alpha1"
	defaultns          = "kudo-system"
	defaultGracePeriod = 10
)

// Options is the configurable options to init
type Options struct {
	// Version is the version of the manager `0.5.0` for example (must NOT include the `v` in `v0.5.0`)
	Version string
	// Namespace to init into (default is kudo-system)
	Namespace string
	// TerminationGracePeriodSeconds defines the termination grace period for a pod
	TerminationGracePeriodSeconds int64
	// Image defines the image to be used
	Image string
}

// NewOptions provides an option struct with defaults
func NewOptions() Options {

	return Options{
		Version:                       version.Get().GitVersion,
		Namespace:                     defaultns,
		TerminationGracePeriodSeconds: defaultGracePeriod,
		Image:                         fmt.Sprintf("kudobuilder/controller:v%v", version.Get().GitVersion),
	}
}

// Install uses Kubernetes client to install KUDO manager.
func Install(client kubernetes.Interface, extClient *apiextensionsclient.Clientset, opts Options) error {
	if err := installPrereqs(client, opts); err != nil {
		return err
	}
	if err := installCrds(extClient); err != nil {
		return err
	}
	if err := installManager(client, opts); err != nil {
		return err
	}
	return nil
}

// Install uses Kubernetes client to install KUDO manager.
func installManager(client kubernetes.Interface, opts Options) error {

	if err := installStatefulSet(client.AppsV1beta2(), opts); err != nil {
		return err
	}

	if err := installService(client.CoreV1(), opts); err != nil {
		return err
	}
	return nil
}

func installStatefulSet(client clientv1beta2.StatefulSetsGetter, opts Options) error {
	ss := generateDeployment(opts)
	_, err := client.StatefulSets(opts.Namespace).Create(ss)
	return err
}

func installService(client corev1.ServicesGetter, opts Options) error {
	s := generateService(opts)
	_, err := client.Services(opts.Namespace).Create(s)
	return err
}

// DeploymentManifests provides a slice of strings for the deployment and service manifest
func DeploymentManifests(opts Options) ([]string, error) {
	s := ManagerService(opts)
	d := ManagerDeployment(opts)

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

// ManagerDeployment provides the KUDO manager deployment manifest for printing
func ManagerDeployment(opts Options) *v1beta2.StatefulSet {
	dep := generateDeployment(opts)

	dep.TypeMeta = metav1.TypeMeta{
		Kind:       "StatefulSet",
		APIVersion: "apps/v1",
	}
	return dep
}

// ManagerService provides the KUDO manager service manifest for printing
func ManagerService(opts Options) *v1.Service {
	svc := generateService(opts)
	svc.TypeMeta = metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
	return svc
}

func generateDeployment(opts Options) *v1beta2.StatefulSet {

	labels := managerLabels()

	secretDefaultMode := int32(420)
	image := opts.Image
	d := &v1beta2.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      "kudo-controller-manager",
			Labels:    labels,
		},
		Spec: v1beta2.StatefulSetSpec{
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			ServiceName: "kudo-controller-manager-service",
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "kudo-manager",
					Containers: []v1.Container{
						{
							Command: []string{"/root/manager"},
							Env: []v1.EnvVar{
								{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "SECRET_NAME", Value: "kudo-webhook-server-secret"},
							},
							Image:           image,
							ImagePullPolicy: "Always",
							Name:            "manager",
							Ports: []v1.ContainerPort{
								// name matters for service
								{ContainerPort: 9876, Name: "webhook-server", Protocol: "TCP"},
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
						{Name: "cert", VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: "kudo-webhook-server-secret", DefaultMode: &secretDefaultMode}}},
					},
				},
			},
		},
	}

	return d
}

func managerLabels() labels.Set {
	labels := generateLabels(map[string]string{"control-plane": "controller-manager", "controller-tools.k8s.io": "1.0"})
	return labels
}

func generateService(opts Options) *v1.Service {
	labels := generateLabels(map[string]string{"control-plane": "controller-manager", "controller-tools.k8s.io": "1.0"})
	s := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      "kudo-controller-manager-service",
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "kudo",
					Port:       443,
					TargetPort: intstr.FromString("webhook-server")},
			},
			Selector: labels,
		},
	}
	return s
}
