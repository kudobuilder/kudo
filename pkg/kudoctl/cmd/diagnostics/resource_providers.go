package diagnostics

// provider functions to be used by resource collectors

import (
	"errors"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
	"k8s.io/kubectl/pkg/describe/versioned"
)

// Object implements runtime.Object and
// metav1.Object interfaces.
// copied from K8 internal type metaRuntimeInterface
type Object interface {
	runtime.Object
	metav1.Object
}

type ObjectWithParent struct {
	Object
	parent *ObjectWithParent
}

type NameHolder struct {
	name string
	parent *ObjectWithParent
}

type NameGetter func() *NameHolder

type NameLister func() []NameHolder

type objectLister func() ([]ObjectWithParent, error)

func listerWithCallback(f objectLister, callback func([]ObjectWithParent)) objectLister {
	return func()([]ObjectWithParent, error) {
		objs, err := f()
		if err == nil {
			callback(objs)
		}
		return objs, err
	}
}

func getterWithCallback(f objectGetter, callback func(*ObjectWithParent)) objectGetter {
	return func()(*ObjectWithParent, error) {
		obj, err := f()
		if err == nil {
			callback(obj)
		}
		return obj, err
	}
}

type objectGetter func() (*ObjectWithParent, error)

type resulter interface {
	result() *ObjectWithParent
}

type resultLister interface {
	result() []ObjectWithParent
}

type resultDumper struct {
	*ObjectWithParent
}

func (r *resultDumper) result() *ObjectWithParent {
	return r.ObjectWithParent
}

type resourceFuncs struct {
	c           *kube.Client
	kc          *kudo.Client
	ns          string
	opts        metav1.ListOptions
	logOpts     corev1.PodLogOptions // TODO: set
	instanceObj *v1beta1.Instance
	root        *ObjectWithParent
}

// instanceObj related resources
func NewInstanceResources(name string, s *env.Settings) (*resourceFuncs, error) {

	kc, err := kudo.NewClient(s.KubeConfig, s.RequestTimeout, s.Validate)
	if err != nil {
		return nil ,err
	}
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil ,err
	}
	instance, err := kc.GetInstance(name, s.Namespace)
	if err != nil {
		return nil ,err
	}
	root := &ObjectWithParent{
		Object: instance,
		parent: nil,
	}
	return &resourceFuncs{
		c:           c,
		kc:          kc,
		ns:          s.Namespace,
		opts:        metav1.ListOptions{LabelSelector: labelKudoOperator + "=" + instance.Labels[labelKudoOperator]},
		logOpts:     corev1.PodLogOptions{},
		instanceObj: instance,
		root:        root,
	}, nil
}

// kudo controller related resources
func NewKudoResources(s *env.Settings) (*resourceFuncs, error) {
	c, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil ,err
	}
	return &resourceFuncs{
		c:        c,
		ns: nsKudoSystem,
		opts:     metav1.ListOptions{LabelSelector: "app=" + appKudoManager},
		logOpts:  corev1.PodLogOptions{},
	}, nil
}

func (f *resourceFuncs) instance() objectGetter {
	return func()(*ObjectWithParent, error) {
		return createObjectWithParent(f.instanceObj, nil, nil)
	}
}

func (f *resourceFuncs) deployments() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.AppsV1().Deployments(f.ns).List(f.opts)
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) events() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.CoreV1().Events(f.ns).List(metav1.ListOptions{})
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) pods() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.CoreV1().Pods(f.ns).List(f.opts)
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) services() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.CoreV1().Services(f.ns).List(f.opts)
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) replicaSets() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.AppsV1().ReplicaSets(f.ns).List(f.opts)
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) statefulSets() objectLister {
	return func() ([]ObjectWithParent, error) {
		obj, err := f.c.KubeClient.AppsV1().StatefulSets(f.ns).List(f.opts)
		return listWithParent(obj, f.root, err)
	}
}

func (f *resourceFuncs) operator(getOpName NameGetter) objectGetter {
	return func() (*ObjectWithParent, error) {
		opName := getOpName()
		obj, err :=  f.kc.GetOperator(opName.name, f.ns)
		return createObjectWithParent(obj, opName.parent, err)
	}
}

func (f *resourceFuncs) operatorVersion(getOvName NameGetter) objectGetter {
	return func() (*ObjectWithParent, error) {
		ovName := getOvName()
		obj, err :=  f.kc.GetOperatorVersion(f.instanceObj.Spec.OperatorVersion.Name, f.ns)
		return createObjectWithParent(obj, ovName.parent, err)
	}
}

// roleBindings - bindings for collected ServiceAccounts for collected Pods
// roleBindings are modified to keep only "relevant" bindings
// TODO: improvement needed: if a pod was collected but its SA failed to, the binding is lost
func (f *resourceFuncs) roleBindings(nameGetters NameLister) objectLister {
	return func() ([]ObjectWithParent, error) {
		ret, err := f.c.KubeClient.RbacV1().RoleBindings(f.ns).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var objs []ObjectWithParent
		// set of service account names
		// TODO: fix filtering: if binding matches SA, ALL the pods should have it
		serviceAccounts := make(map[string]*ObjectWithParent)
		for _, nh := range nameGetters() {
			serviceAccounts[nh.name] = nh.parent
		}
		for i := range ret.Items {
			item := &ret.Items[i]
			// filtering subjects "in place" by service account names
			j := 0
			for _, subject := range item.Subjects {
				if sa, ok := serviceAccounts[subject.Name]; ok && subject.Kind == "ServiceAccount" {
					item.Subjects[j] = subject
					j++
					// assume that if we can't set GVK a cluster role, things are weird enough to quit
					objs = append(objs, ObjectWithParent{
						Object: item,
						parent: sa,
					})
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
func (f *resourceFuncs) clusterRoleBindings(nameGetters NameLister) objectLister {
	return func() ([]ObjectWithParent, error) {
		ret, err := f.c.KubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var objs []ObjectWithParent
		// TODO: fix filtering: if binding matches SA, ALL the pods should have it
		// set of service account names
		serviceAccounts := make(map[string]*ObjectWithParent)
		for _, nh := range nameGetters() {
			serviceAccounts[nh.name] = nh.parent
		}
		for i := range ret.Items {
			item := &ret.Items[i]
			// filtering subjects "in place" by service account names
			j := 0
			for _, subject := range item.Subjects {
				if sa, ok := serviceAccounts[subject.Name]; ok && subject.Kind == "ServiceAccount" {
					item.Subjects[j] = subject
					j++
					// assume that if we can't set GVK a cluster role, things are weird enough to quit
					objs = append(objs, ObjectWithParent{
						Object: item,
						parent: sa,
					})
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

func (f *resourceFuncs) serviceAccounts(nameGetters NameLister) objectLister {
	return objectGetterProviderFn(f.serviceAccount).toListerProviderFn(nameGetters)()
}

func (f *resourceFuncs) serviceAccount(getSaName NameGetter) objectGetter {
	return func()(*ObjectWithParent, error) {
		saName := getSaName()
		obj, err := f.c.KubeClient.CoreV1().ServiceAccounts(f.ns).Get(saName.name, metav1.GetOptions{})
		return createObjectWithParent(obj, saName.parent, err)
	}
}

type objectGetterProviderFn func(NameGetter) objectGetter

type objectListerProviderFn func() objectLister

func (fn objectGetterProviderFn) toListerProviderFn(nameHolders NameLister) objectListerProviderFn {
	return func() objectLister {
		return func()([]ObjectWithParent, error) {
			var ret []ObjectWithParent
			var err *MultiError
			for _, nh := range nameHolders() {
				obj, e := fn(nameDumper(&nh))()
				if e == nil {
					e = SetGVKFromScheme(obj.Object)
				}
				err = AppendError(err, e)
				if e == nil {
					ret = append(ret, *obj)
				}
			}
			if err != nil {
				return ret, err
			}
			return ret, nil
		}
	}
}

func (f *resourceFuncs) clusterRole(getCRoleName NameGetter) objectGetter {
	return func()(*ObjectWithParent, error) {
		roleName := getCRoleName()
		obj, err := f.c.KubeClient.RbacV1().ClusterRoles().Get(roleName.name, metav1.GetOptions{})
		return createObjectWithParent(obj, roleName.parent, err)
	}
}

func (f *resourceFuncs) clusterRoles(nameGetters NameLister) objectLister {
	return objectGetterProviderFn(f.clusterRole).toListerProviderFn(nameGetters)()
}

func (f *resourceFuncs) role(getCRoleName NameGetter) objectGetter {
	return func()(*ObjectWithParent, error) {
		roleName := getCRoleName()
		obj, err := f.c.KubeClient.RbacV1().Roles(f.ns).Get(roleName.name, metav1.GetOptions{})
		return createObjectWithParent(obj, roleName.parent, err)
	}
}

func (f *resourceFuncs) roles(nameGetters NameLister) objectLister {
	return objectGetterProviderFn(f.role).toListerProviderFn(nameGetters)()
}

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

func listWithParent(obj runtime.Object, parent *ObjectWithParent, e error) ([]ObjectWithParent, error) {
	objs, err := listOf(obj, e)
	if err != nil {
		return nil, err
	}
	ret  := make([]ObjectWithParent, len(objs))
	for i := 0; i < len(objs); i++ {
		ret[i] = ObjectWithParent{
			Object: objs[i].(Object),
			parent: parent,
		}
	}
	return ret, nil
}

func createObjectWithParent(obj runtime.Object, parent *ObjectWithParent, e error) (*ObjectWithParent, error) {
	if e != nil {
		return nil, e
	}
	return &ObjectWithParent{
		Object: obj.(Object),
		parent: parent,
	}, nil
}