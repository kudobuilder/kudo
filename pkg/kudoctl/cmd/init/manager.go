package init

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
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/version"
)

//Defines the deployment of the KUDO manager and it's service definition.

const (
	group                 = "kudo.dev"
	crdVersion            = "v1beta1"
	defaultns             = "kudo-system"
	defaultGracePeriod    = 10
	defaultServiceAccount = "kudo-manager"
)

// Options is the configurable options to init
type Options struct {
	// Version is the version of the manager `0.5.0` for example (must NOT include the `v` in `v0.5.0`)
	Version string
	// namespace to init into (default is kudo-system)
	Namespace string
	// TerminationGracePeriodSeconds defines the termination grace period for a pod
	TerminationGracePeriodSeconds int64
	// Image defines the image to be used
	Image string
	// Enable validation
	Webhooks       []string
	ServiceAccount string
}

func (o Options) webhooksEnabled() bool {
	return len(o.Webhooks) != 0
}

// NewOptions provides an option struct with defaults
func NewOptions(v string, ns string, sa string, webhooks []string) Options {

	if v == "" {
		v = version.Get().GitVersion
	}
	if ns == "" {
		ns = defaultns
	}

	if sa == "" {
		sa = defaultServiceAccount
	}

	return Options{
		Version:                       v,
		Namespace:                     ns,
		TerminationGracePeriodSeconds: defaultGracePeriod,
		Image:                         fmt.Sprintf("kudobuilder/controller:v%v", v),
		Webhooks:                      webhooks,
		ServiceAccount:                sa,
	}
}

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts Options, crdOnly bool) error {

	if err := installCrds(client.ExtClient); err != nil {
		return err
	}
	clog.Printf("✅ installed crds")
	if crdOnly {
		return nil
	}
	if err := installPrereqs(client.KubeClient, opts); err != nil {
		return err
	}
	clog.Printf("✅ installed service accounts and other requirements for controller to run")

	if opts.webhooksEnabled() {
		if err := installInstanceValidatingWebhook(client.KubeClient, client.DynamicClient, opts.Namespace); err != nil {
			return err
		}
		clog.Printf("✅ installed webhook")
	}

	if err := installManager(client.KubeClient, opts); err != nil {
		return err
	}
	clog.Printf("✅ installed kudo controller")
	return nil
}

// Install uses Kubernetes client to install KUDO.
func installManager(client kubernetes.Interface, opts Options) error {
	if err := installStatefulSet(client.AppsV1(), opts); err != nil {
		return err
	}

	if err := installService(client.CoreV1(), opts); err != nil {
		return err
	}
	return nil
}

func installStatefulSet(client appsv1client.StatefulSetsGetter, opts Options) error {
	ss := generateDeployment(opts)
	_, err := client.StatefulSets(opts.Namespace).Create(ss)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("statefulset %v already exists", ss.Name)
		return nil
	}
	return err
}

func installService(client corev1.ServicesGetter, opts Options) error {
	s := generateService(opts)
	_, err := client.Services(opts.Namespace).Create(s)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service %v already exists", s.Name)
		return nil
	}

	return err
}

// ManagerManifests provides a slice of strings for the deployment and service manifest
func ManagerManifests(opts Options) ([]string, error) {
	s := managerService(opts)
	d := managerDeployment(opts)

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

// managerDeployment provides the KUDO manager deployment manifest for printing
func managerDeployment(opts Options) *appsv1.StatefulSet {
	dep := generateDeployment(opts)

	dep.TypeMeta = metav1.TypeMeta{
		Kind:       "StatefulSet",
		APIVersion: "apps/v1",
	}
	return dep
}

// managerService provides the KUDO manager service manifest for printing
func managerService(opts Options) *v1.Service {
	svc := generateService(opts)
	svc.TypeMeta = metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}
	return svc
}

func generateDeployment(opts Options) *appsv1.StatefulSet {

	labels := managerLabels()

	secretDefaultMode := int32(420)
	image := opts.Image
	d := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      "kudo-controller-manager",
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			ServiceName: "kudo-controller-manager-service",
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: opts.ServiceAccount,
					Containers: []v1.Container{

						{

							Command: []string{"/root/manager"},
							Env: []v1.EnvVar{
								{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "SECRET_NAME", Value: "kudo-webhook-server-secret"},
								{Name: "ENABLE_WEBHOOKS", Value: strconv.FormatBool(opts.webhooksEnabled())},
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
