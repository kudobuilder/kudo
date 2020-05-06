package diagnostics

// provider functions to be used by resource collectors

import (
	//"errors"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/client-go/rest"
	//"k8s.io/kubectl/pkg/describe"
	//"k8s.io/kubectl/pkg/describe/versioned"
)

// TODO: handle missing provider
type resourceProviderFactory struct {
	r       *resourceFuncs
	getters map[string]objectGetterProviderFn
	listers map[string]objectListerProviderFn
}

func newResourceProviderFactory(r *resourceFuncs) *resourceProviderFactory {
	getters := map[string]objectGetterProviderFn{
		"role":           r.role,
		"clusterrole":    r.clusterRole,
		"serviceaccount": r.serviceAccount,
		"operator": r.operator,
		"operatorversion": r.operatorVersion,
	}
	listers := map[string]objectListerProviderFn{
		"instance": r.instances,
		"deployment":  r.deployments,
		"statefulset": r.statefulSets,
		"replicaset":  r.replicaSets,
		"pod":         r.pods,
		"service":     r.services,
		"event":       r.events,
	}
	return &resourceProviderFactory{
		r:       r,
		getters: getters,
		listers: listers,
	}
}

func (f *resourceProviderFactory) Lister(kind string) objectLister {
	return f.listers[strings.ToLower(kind)]() // TODO: handle missing
}

func (f *resourceProviderFactory) MultiGetter(kind string, names []string) objectLister {
	k := strings.ToLower(kind)
	switch k {
	case "rolebinding":
		return f.r.roleBindings(names)
	case "clusterrolebinding":
		return f.r.clusterRoleBindings(names)
	}
	return f.getters[kind].toListerProviderFn(names)()
}

func (f *resourceProviderFactory) getter(kind, name string) objectGetter {
	return f.getters[strings.ToLower(kind)](name)
}

// Object implements runtime.Object and
// metav1.Object interfaces.
// copied from K8 internal type metaRuntimeInterface
type Object interface {
	runtime.Object
	metav1.Object
}

type objectLister func() ([]Object, error)

type objectGetter func() (Object, error)

type resourceFuncs struct {
	c           *kube.Client
	kc          *kudo.Client
	ns          string
	opts        metav1.ListOptions
	logOpts     corev1.PodLogOptions // TODO: set
	instanceObj *v1beta1.Instance
	root        Object
}

// instanceObj related resources
func NewInstanceResources(name string, s *env.Settings) (*resourceFuncs, error) {

	kc, err := kudo.NewClient(s.KubeConfig, s.RequestTimeout, s.Validate)
	if err != nil {
		return nil, err
	}
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil, err
	}
	instance, err := kc.GetInstance(name, s.Namespace)
	if err != nil {
		return nil, err
	}

	return &resourceFuncs{
		c:           c,
		kc:          kc,
		ns:          s.Namespace,
		opts:        metav1.ListOptions{LabelSelector: labelKudoOperator + "=" + instance.Labels[labelKudoOperator]},
		logOpts:     corev1.PodLogOptions{},
		instanceObj: instance,
		root:        instance,
	}, nil
}

// kudo controller related resources
func NewKudoResources(s *env.Settings) (*resourceFuncs, error) {
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil, err
	}
	return &resourceFuncs{
		c:       c,
		ns:      nsKudoSystem,
		opts:    metav1.ListOptions{LabelSelector: "app=" + appKudoManager},
		logOpts: corev1.PodLogOptions{},
	}, nil
}

// TODO: get rid of
func (f *resourceFuncs) instances() objectLister {
	return func() ([]Object, error) {
		if f.instanceObj == nil {
			return nil, nil
		}
		return []Object{f.instanceObj}, nil
	}
}

func (f *resourceFuncs) instance() objectGetter {
	return func() (Object, error) {
		return f.instanceObj, nil
	}
}

func (f *resourceFuncs) deployments() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.AppsV1().Deployments(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) events() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.CoreV1().Events(f.ns).List(metav1.ListOptions{})
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) pods() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.CoreV1().Pods(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) services() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.CoreV1().Services(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) replicaSets() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.AppsV1().ReplicaSets(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) statefulSets() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.AppsV1().StatefulSets(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) operator(opName string) objectGetter {
	return func() (Object, error) {
		obj, err := f.kc.GetOperator(opName, f.ns)
		return obj, err
	}
}

func (f *resourceFuncs) operatorVersion(ovName string) objectGetter {
	return func() (Object, error) {
		obj, err := f.kc.GetOperatorVersion(f.instanceObj.Spec.OperatorVersion.Name, f.ns)
		return obj, err
	}
}

// roleBindings - bindings for collected ServiceAccounts for collected Pods
// roleBindings are modified to keep only "relevant" bindings
// TODO: improvement needed: if a pod was collected but its SA failed to, the binding is lost
func (f *resourceFuncs) roleBindings(saNames []string) objectLister {
	return func() ([]Object, error) {
		ret, err := f.c.KubeClient.RbacV1().RoleBindings(f.ns).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var objs []Object
		// set of service account names
		serviceAccounts := make(map[string]struct{})
		for _, saName := range saNames {
			serviceAccounts[saName] = struct{}{}
		}
		for i := range ret.Items {
			item := &ret.Items[i]
			// filtering subjects "in place" by service account names
			j := 0
			for _, subject := range item.Subjects {
				if _, ok := serviceAccounts[subject.Name]; ok && subject.Kind == "ServiceAccount" {
					item.Subjects[j] = subject
					j++
					// assume that if we can't set GVK a cluster role, things are weird enough to quit
					objs = append(objs, item)
				}
			}
			if j > 0 {
				if err = SetGVKFromScheme(item); err != nil {
					return nil, err
				}
				item.Subjects = item.Subjects[:j]
			}
		}
		return objs, nil
	}
}

// clusterRoleBindings - bindings for collected ServiceAccounts for collected Pods
// clusterRoleBindings are modified to keep only "relevant" bindings
// TODO: improvement needed: if a pod was collected but its SA failed to, the binding is lost
func (f *resourceFuncs) clusterRoleBindings(saNames []string) objectLister {
	return func() ([]Object, error) {
		ret, err := f.c.KubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var objs []Object
		// set of service account names
		serviceAccounts := make(map[string]struct{})
		for _, saName := range saNames {
			serviceAccounts[saName] = struct{}{}
		}
		for i := range ret.Items {
			item := &ret.Items[i]
			// filtering subjects "in place" by service account names
			j := 0
			for _, subject := range item.Subjects {
				if _, ok := serviceAccounts[subject.Name]; ok && subject.Kind == "ServiceAccount" {
					item.Subjects[j] = subject
					j++
					// assume that if we can't set GVK a cluster role, things are weird enough to quit
					objs = append(objs, item)
				}
			}
			if j > 0 {
				if err = SetGVKFromScheme(item); err != nil {
					return nil, err
				}
				item.Subjects = item.Subjects[:j]
			}
		}
		return objs, nil
	}
}

func (f *resourceFuncs) serviceAccount(saName string) objectGetter {
	return func() (Object, error) {
		obj, err := f.c.KubeClient.CoreV1().ServiceAccounts(f.ns).Get(saName, metav1.GetOptions{})
		return obj, err
	}
}

func (f *resourceFuncs) clusterRole(roleName string) objectGetter {
	return func() (Object, error) {
		obj, err := f.c.KubeClient.RbacV1().ClusterRoles().Get(roleName, metav1.GetOptions{})
		return obj, err
	}
}

func (f *resourceFuncs) role(roleName string) objectGetter {
	return func() (Object, error) {
		obj, err := f.c.KubeClient.RbacV1().Roles(f.ns).Get(roleName, metav1.GetOptions{})
		return obj, err
	}
}

type (
	objectGetterProviderFn func(string) objectGetter
	objectListerProviderFn func() objectLister
)

func (fn objectGetterProviderFn) toListerProviderFn(names []string) objectListerProviderFn {
	return func() objectLister {
		return func() ([]Object, error) {
			var ret []Object
			var err *MultiError
			for _, name := range names {
				obj, e := fn(name)()
				if e == nil && !isKudoCR(obj){
					e = SetGVKFromScheme(obj)
				}
				err = AppendError(err, e)
				if e == nil {
					ret = append(ret, obj)
				}
			}
			if err != nil {
				return ret, err
			}
			return ret, nil
		}
	}
}

func isKudoCR(o Object) bool {
	kind := o.GetObjectKind().GroupVersionKind().Kind
	return kind == "Instance" || kind == "Operator" || kind == "OperatorVersion"
}

/*
// TODO: logs and descriptions: fix and uncomment
func (f *resourceFuncs) logs(podNames NameLister) func() ([]logHolder, error) {
	return func() ([]logHolder, error) {
		var ret []logHolder
		for _, ph := range podNames() {
			log, err := f.c.KubeClient.CoreV1().Pods(f.ns).GetLogs(ph.name, &f.logOpts).Stream()
			if err != nil {
				return nil, err
			}
			ret = append(ret, logHolder{
				logStream: log,
				t:         LogInfoType,
				kind:      "pod",
				podParent: ph.parent,
				podName: ph.name,
				nameSpace: f.ns,
			})
		}
		return ret, nil
	}
}

func descriptions(describedCollector resultLister, config *rest.Config) func() ([]descriptionHolder, error) {
	return func() ([]descriptionHolder, error) {
		var ret []descriptionHolder
		var err *MultiError
		for _, p := range describedCollector.result() {
			dh, e := description(&resultDumper{&p}, config)()
			err = AppendError(err, e)
			if e == nil {
				ret = append(ret, *dh)
			}
		}
		if err != nil {
			return ret, err
		}
		return ret, nil
	}
}


func description(descriptionCollector resulter, config *rest.Config) func() (*descriptionHolder, error) {
	return func() (*descriptionHolder, error) {
		p := descriptionCollector.result()
		describer, ok := versioned.DescriberFor(p.GetObjectKind().GroupVersionKind().GroupKind(), config)
		if !ok {
			return nil, errors.New("describer for " + p.GetObjectKind().GroupVersionKind().GroupKind().String() + " not found" )
		}
		desc, err := describer.Describe(p.GetNamespace(), p.GetName(), describe.DescriberSettings{ShowEvents: true})
		if err != nil {
			return nil, err
		}

		return &descriptionHolder{
			desc:   desc,
			t:      DescribeInfoType,
			kind:   p.GetObjectKind().GroupVersionKind().Kind,
			Object: p,
		}, nil
	}
}
*/
// listOf - convert a runtime.Object representing a list to a slice and set GVKs
func listOf(obj runtime.Object, e error) ([]Object, error) {
	if e != nil {
		return nil, e
	}
	objs, err := meta.ExtractList(obj)
	if err != nil {
		return nil, err
	}
	ret := make([]Object, len(objs))
	var ok bool
	for i := 0; i < len(objs); i++ {
		err = SetGVKFromScheme(objs[i])
		if err != nil {
			return nil, err
		}
		if ret[i], ok = objs[i].(Object); !ok {
			return nil, err
		}
	}
	return ret, nil
}

