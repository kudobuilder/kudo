package diagnostics

// provider functions to be used by resource collectors

import (
	"errors"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
	"k8s.io/kubectl/pkg/describe/versioned"
)

// TODO: we need a parent object for a flexible hierarchical structure
// e.g: if service accounts are collected for pods, we'd like to have
// pods

// Object implements runtime.Object and
// metav1.Object interfaces.
// copied from K8 internal type metaRuntimeInterface
type Object interface {
	runtime.Object
	metav1.Object
}

type objectLister func() ([]Object, error)

type objectGetter func() (Object, error)

type resulter interface {
	result() Object
}

type resultLister interface {
	result() []Object
}

type resultDumper struct {
	Object
}

func (r *resultDumper) result() Object {
	return r.Object
}

type resourceFuncs struct {
	c        *kube.Client
	kc       *kudo.Client
	ns       string
	opts     metav1.ListOptions
	logOpts  corev1.PodLogOptions // TODO: set
	instance *v1beta1.Instance
}

func (f *resourceFuncs) deployments() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.AppsV1().Deployments(f.ns).List(f.opts)
		return listOf(obj, err)
	}
}

func (f *resourceFuncs) events() objectLister {
	return func() ([]Object, error) {
		obj, err := f.c.KubeClient.CoreV1().Events(f.ns).List(f.opts)
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

func (f *resourceFuncs) operators(operatorVersion resulter) objectGetter {
	return func() (Object, error) {
		ov := operatorVersion.result().(*v1beta1.OperatorVersion)
		return f.kc.GetOperator(ov.Spec.Operator.Name, f.ns)
	}
}

func (f *resourceFuncs) operatorVersions() objectGetter {
	return func() (Object, error) {
		return f.kc.GetOperatorVersion(f.instance.Spec.OperatorVersion.Name, f.ns)
	}
}

// roleBindings - bindings for collected ServiceAccounts for collected Pods
// TODO: improvement needed: if a pod was collected but its SA failed to, the binding is lost
func (f *resourceFuncs) roleBindings(serviceAccounts resultLister) objectLister {
	return func() ([]Object, error) {
		ret, err := f.c.KubeClient.RbacV1().RoleBindings(f.ns).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		// set of service account names
		saExist := make(map[string]bool)
		for _, sa := range serviceAccounts.result() {
			saExist[sa.GetName()] = true
		}
		// filtering "in place" by service account names:
		// Items having at least one Subject referencing one of the provided Service Account,
		j := 0
		for _, item := range ret.Items {
			for _, subject := range item.Subjects {
				if subject.Kind == "ServiceAccount" && saExist[subject.Name] {
					ret.Items[j] = item
					j++
					break
				}
			}
		}
		ret.Items = ret.Items[:j]
		return listOf(ret, nil)
	}
}

// clusterRoleBindings - bindings for collected ServiceAccounts for collected Pods
// TODO: improvement needed: if a pod was collected but its sa failed to, the binding is lost
func (f *resourceFuncs) clusterRoleBindings(serviceAccounts resultLister) objectLister {
	return func() ([]Object, error) {
		ret, err := f.c.KubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		// set of service account names
		saExist := make(map[string]bool)
		for _, sa := range serviceAccounts.result() {
			saExist[sa.GetName()] = true
		}
		// filtering "in place" by service account names:
		// Items having at least one Subject referencing one of the provided Service Account,
		j := 0
		for _, item := range ret.Items {
			for _, subject := range item.Subjects {
				if subject.Kind == "ServiceAccount" && saExist[subject.Name] {
					ret.Items[j] = item
					j++
					break
				}
			}
		}
		ret.Items = ret.Items[:j]
		return listOf(ret, nil)
	}
}

// service accounts for collected pods
// service account used by a KUDO-labeled pod may not necessarily be created by KUDO
// TODO: duplicated code with roles and clusterRoles
func (f *resourceFuncs) serviceAccounts(pods resultLister) objectLister {
	return func() ([]Object, error) {
		var ret []Object
		var err *multiError
		for _, p := range pods.result() {
			pod := p.(*corev1.Pod)
			obj, e := f.c.KubeClient.CoreV1().ServiceAccounts(f.ns).Get(pod.Spec.ServiceAccountName, metav1.GetOptions{})
			if e == nil {
				e = setGVKFromScheme(obj)
			}
			err = appendError(err, e)
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

func (f *resourceFuncs) clusterRoles(bindingsCollector resultLister) objectLister {
	return func() ([]Object, error) {
		var ret []Object
		var err *multiError
		for _, b := range bindingsCollector.result() {
			binding := b.(*rbacv1.ClusterRoleBinding)
			obj, e := f.c.KubeClient.RbacV1().ClusterRoles().Get(binding.RoleRef.Name, metav1.GetOptions{})
			if e == nil {
				e = setGVKFromScheme(obj)
			}
			err = appendError(err, e)
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

func (f *resourceFuncs) roles(bindingsCollector resultLister) objectLister {
	return func() ([]Object, error) {
		var ret []Object
		var err *multiError
		for _, b := range bindingsCollector.result() {
			binding := b.(*rbacv1.RoleBinding)
			obj, e := f.c.KubeClient.RbacV1().Roles(f.ns).Get(binding.RoleRef.Name, metav1.GetOptions{})
			if e == nil {
				e = setGVKFromScheme(obj)
			}
			err = appendError(err, e)
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

func (f *resourceFuncs) logs(podsCollector resultLister) func() ([]logHolder, error) {
	return func() ([]logHolder, error) {
		var ret []logHolder
		for _, p := range podsCollector.result() {
			pod := p.(*corev1.Pod)
			log, err := f.c.KubeClient.CoreV1().Pods(f.ns).GetLogs(pod.Name, &f.logOpts).Stream()
			if err != nil {
				return nil, err
			}
			ret = append(ret, logHolder{
				logStream: log,
				t:         LogInfoType,
				kind:      "pod",
				Object:    pod,
			})
		}
		return ret, nil
	}
}

func descriptions(describedCollector resultLister, config *rest.Config) func() ([]describeHolder, error) {
	return func() ([]describeHolder, error) {
		var ret []describeHolder
		var err *multiError
		for _, p := range describedCollector.result() {
			dh, e := description(&resultDumper{p}, config)()
			err = appendError(err, e)
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

func description(describedCollector resulter, config *rest.Config) func() (*describeHolder, error) {
	return func() (*describeHolder, error) {
		p := describedCollector.result()
		describer, ok := versioned.DescriberFor(p.GetObjectKind().GroupVersionKind().GroupKind(), config)
		if !ok {
			return nil, errors.New("describer for " + p.GetObjectKind().GroupVersionKind().GroupKind().String() + " not found" )
		}
		desc, err := describer.Describe(p.GetNamespace(), p.GetName(), describe.DescriberSettings{ShowEvents: true})
		if err != nil {
			return nil, err
		}

		return &describeHolder{
			desc:   desc,
			t:      DescribeInfoType,
			kind:   p.GetObjectKind().GroupVersionKind().Kind,
			Object: p,
		}, nil
	}
}

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
		err = setGVKFromScheme(objs[i])
		if err != nil {
			return nil, err
		}
		if ret[i], ok = objs[i].(Object); !ok {
			return nil, err
		}
	}
	return ret, nil
}
