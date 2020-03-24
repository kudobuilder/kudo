package diagnostics

// provider functions to be used by resource collectors

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type resourceFuncs struct {
	c        *kube.Client
	kc       *kudo.Client
	ns       string
	opts     metav1.ListOptions
	instance *v1beta1.Instance
}

func (f *resourceFuncs) deployments() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.AppsV1().Deployments(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) events() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.CoreV1().Events(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) pods() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.CoreV1().Pods(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) services() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.CoreV1().Services(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) replicaSets() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.AppsV1().ReplicaSets(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) statefulSets() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			return f.c.KubeClient.AppsV1().StatefulSets(f.ns).List(f.opts)
		})
	}
}

func (f *resourceFuncs) operators(operatorVersion *resourceCollector) objectGetter {
	return func() (runtime.Object, error) {
		ov := operatorVersion.result().(*v1beta1.OperatorVersion)
		return f.kc.GetOperator(ov.Spec.Operator.Name, f.ns)
	}
}

func (f *resourceFuncs) operatorVersions() objectGetter {
	return func() (runtime.Object, error) {
		return f.kc.GetOperatorVersion(f.instance.Spec.OperatorVersion.Name, f.ns)
	}
}

// TODO: filter properly, get rid of the hardcoded appKudoManager
func (f *resourceFuncs) roleBindings() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			ret, err := f.c.KubeClient.RbacV1().RoleBindings(f.ns).List(metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			// TODO: do not filter ret.Items, filter "our" array of runtime objects (what listOf returns)
			j := 0
			for _, item := range ret.Items {
				for _, subject := range item.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Name == appKudoManager {
						//ret = append(ret, item) // TODO: do we by chance need DeepCopy() here
						ret.Items[j] = item
						j++
						break
					}
				}
			}
			ret.Items = ret.Items[:j]
			return ret, nil
		})
	}
}

// TODO: filter properly, get rid of the hardcoded appKudoManager
func (f *resourceFuncs) clusterRoleBindings() objectLister {
	return func() ([]runtime.Object, error) {
		return listOf(func() (runtime.Object, error) {
			ret, err := f.c.KubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			// TODO: do not filter ret.Items, filter "our" array of runtime objects
			j := 0
			for _, item := range ret.Items {
				for _, subject := range item.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Name == appKudoManager {
						//ret = append(ret, item) // TODO: do we by chance need DeepCopy() here
						ret.Items[j] = item
						j++
						break
					}
				}
			}
			ret.Items = ret.Items[:j]
			return ret, nil
		})
	}
}

// TODO: set GVK!!!
func (f *resourceFuncs) clusterRoles(bindingsCollector *resourceListCollector) objectLister {
	return func() ([]runtime.Object, error) {
		var ret []runtime.Object
		for _, b := range bindingsCollector.result() {
			binding := b.(*rbacv1.ClusterRoleBinding)
			role, err := f.c.KubeClient.RbacV1().ClusterRoles().Get(binding.RoleRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err // TODO: collect errors here
			}
			err = setGVKFromScheme(role)
			if err != nil {
				return nil, err //TODO: unreal, but what to do here?
			}
			ret = append(ret, role)
		}
		return ret, nil
	}
}

func (f *resourceFuncs) roles(bindingsCollector *resourceListCollector) objectLister {
	return func() ([]runtime.Object, error) {
		var ret []runtime.Object
		for _, b := range bindingsCollector.result() {
			binding := b.(*rbacv1.RoleBinding)
			role, err := f.c.KubeClient.RbacV1().Roles(f.ns).Get(binding.RoleRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err // TODO: collect errors here
			}
			err = setGVKFromScheme(role)
			if err != nil {
				return nil, err //TODO: unreal, but what to do here?
			}
			ret = append(ret, role)
		}
		return ret, nil
	}
}

func listOf(lf func() (runtime.Object, error)) ([]runtime.Object, error) {
	obj, err := lf()
	if err != nil {
		return nil, err
	}
	objs, err := meta.ExtractList(obj)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(objs); i++ {
		err = setGVKFromScheme(objs[i])
		if err != nil {
			return nil, err //TODO: unreal, but what to do here?
		}
	}
	return objs, nil
}
