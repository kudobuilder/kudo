/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package componentversion

import (
	"testing"
	"time"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	frameworksv1beta1 "github.com/kudobuilder/kudo/pkg/apis/frameworks/v1beta1"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
var depKey = types.NamespacedName{Name: "foo-deployment", Namespace: "default"}

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &frameworksv1beta1.ComponentVersion{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create the ComponentVersion object and expect the Reconcile and Deployment to be created
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	deploy := &appsv1.Deployment{}
	g.Eventually(func() error { return c.Get(context.TODO(), depKey, deploy) }, timeout).
		Should(gomega.Succeed())

	// Delete the Deployment and expect Reconcile to be called for Deployment deletion
	g.Expect(c.Delete(context.TODO(), deploy)).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), depKey, deploy) }, timeout).
		Should(gomega.Succeed())

	// Manually delete Deployment since GC isn't enabled in the test control plane
	g.Eventually(func() error { return c.Delete(context.TODO(), deploy) }, timeout).
		Should(gomega.MatchError("deployments.apps \"foo-deployment\" not found"))

}

func TestDynamicCRD(t *testing.T) {
	//when adding a zk component and componentversion, we get a new CRD
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())
	//remove next line
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	comp := &frameworksv1beta1.Component{ObjectMeta: metav1.ObjectMeta{Name: "zookeeper", Namespace: "default"}}
	crd := apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "component.kudo.sh",
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group: "kudo.sh",
			// Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
			// 	{
			// 		Name:    "v1beta1",
			// 		Served:  true,
			// 		Storage: true,
			// 	},
			// },
			Scope: apiextv1beta1.NamespaceScoped,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural:     "zookeepers",
				Singular:   "zookeeper",
				Kind:       "Zookeeper",
				ShortNames: []string{"zk"},
			},
		},
	}

	comp.Spec.CustomResourceDefinition = crd
	//create component
	err = c.Create(context.TODO(), comp)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), comp)

	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	ver := &frameworksv1beta1.ComponentVersion{ObjectMeta: metav1.ObjectMeta{Name: "v3.2.1", Namespace: "default"}}
	ver.Spec.Version = "v3.2.1"

	err = c.Create(context.TODO(), ver)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), ver)

	ver2 := &frameworksv1beta1.ComponentVersion{ObjectMeta: metav1.ObjectMeta{Name: "v3.2.1", Namespace: "default"}}
	ver2.Spec.Version = "v4"

	err = c.Create(context.TODO(), ver2)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), ver2)

	/*
		apiVersion: frameworks.kudo.sh/v1beta1
		kind: Component
		metadata:
		  labels:
		    controller-tools.k8s.io: "1.0"
		  name: zookeeper-component
		spec:
		  # Add fields here
		  customresourcedefinition:
		    metadata:
		      name: zookeepers.zookeeper.kudo.sh
		    spec:
		      group: zookeeper.kudo.sh
		      versions:
		        - name: v1beta1
		          served: true
		          storage: true
		      scope: Namespaced
		      names:
		        plural: zookeepers
		        singular: zookeeper
		        kind: Zookeeper
		        shortNames:
		          - zk
		---
		apiVersion: frameworks.kudo.sh/v1beta1
		kind: ComponentVersion
		metadata:
		  labels:
		    controller-tools.k8s.io: "1.0"
		  name: zookeeper-1.0
		spec:
		  # Add fields here
		  version: "1.0"
		  parameters:
		    memory:
		      description: Amount of memory to provide Zookeeper pods
		      default: "1Gi"
		      trigger: update
		    cpus:
		      description: Amount of cpu to provide Zooekeper pods
		      default: "0.25"
	*/

}
