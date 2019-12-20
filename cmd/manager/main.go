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

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	apiextenstionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kudobuilder/kudo/pkg/apis"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/operator"
	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"
	"github.com/kudobuilder/kudo/pkg/version"
)

// parseSyncPeriod determines the minimum frequency at which watched resources are reconciled.
// If the variable is present in the environment the duration is returned.
func parseSyncPeriod() (*time.Duration, error) {
	if val, ok := os.LookupEnv("KUDO_SYNCPERIOD"); ok {
		sync, err := time.ParseDuration(val)
		if err != nil {
			return nil, err
		}
		return &sync, nil
	}
	return nil, nil
}

func main() {
	// Get version of KUDO
	log.Printf("KUDO Version: %#v", version.Get())

	// create new controller-runtime manager

	syncPeriod, err := parseSyncPeriod()
	if err != nil {
		log.Printf("unable to parse manager sync period variable: %v", err)
		os.Exit(1)
	}

	if syncPeriod != nil {
		log.Print(fmt.Sprintf("setting up manager, sync-period is %v", syncPeriod))
	} else {
		log.Print("setting up manager")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		CertDir:    "/tmp/cert",
		SyncPeriod: syncPeriod,
	})
	if err != nil {
		log.Printf("unable to start manager: %v", err)
		os.Exit(1)
	}

	log.Print("Registering Components")

	log.Print("setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("unable to add APIs to scheme: %v", err)
	}

	if err := apiextenstionsv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("unable to add extension APIs to scheme: %v", err)
	}

	// Setup all Controllers

	log.Print("Setting up operator controller")
	err = (&operator.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("unable to register operator controller to the manager: %v", err)
		os.Exit(1)
	}

	log.Print("Setting up operator version controller")
	err = (&operatorversion.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("unable to register operator controller to the manager: %v", err)
		os.Exit(1)
	}

	log.Print("Setting up instance controller")
	err = (&instance.Reconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("instance-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("unable to register instance controller to the manager: %v", err)
		os.Exit(1)
	}

	if strings.ToLower(os.Getenv("ENABLE_WEBHOOKS")) == "true" {
		log.Printf("Setting up webhooks")

		if err := registerWebhook("/validate", &v1beta1.Instance{}, &webhook.Admission{Handler: &v1beta1.InstanceAdmission{}}, mgr); err != nil {
			log.Printf("unable to create instance validation webhook: %v", err)
			os.Exit(1)
		}

		// Add more webhooks below using the above registerWebhook method
	}

	// Start the KUDO manager
	log.Print("Starting KUDO manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Printf("unable to run the manager: %v", err)
		os.Exit(1)
	}
}

// registerWebhook method registers passed webhook using a give prefix (e.g. "/validate") and runtime object
// (e.g. v1beta1.Instance) to generate a webhook path e.g. "/validate-kudo-dev-v1beta1-instances". Webhook
// has to implement http.Handler interface (see v1beta1.InstanceAdmission for an example)
func registerWebhook(prefix string, obj runtime.Object, hook http.Handler, mgr manager.Manager) error {
	path, err := webhookPath(prefix, obj, mgr)
	if err != nil {
		return fmt.Errorf("failed to generate webhook path: %v", err)
	}

	// an already handled path is likely a misconfiguration that should be fixed. failing loud and proud
	if isAlreadyHandled(path, mgr) {
		return fmt.Errorf("webhook path %s is already handled", path)
	}

	mgr.GetWebhookServer().Register(path, hook)

	return nil
}

// webhookPath method generates a unique path for a given prefix and object GVK of the form:
// /validate-kudo-dev-v1beta1-instances
// Not that object version is included in the path since different object versions can have
// different webhooks. If the strategy to generate this path changes we should update init
// code and webhook setup. Right now this is in sync how controller-runtime generates these paths
func webhookPath(prefix string, obj runtime.Object, mgr manager.Manager) (string, error) {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return "", err
	}

	// if the strategy to generate this path changes we should update init code and webhook setup
	// right now this is in sync how controller-runtime generates these paths
	return prefix + "-" + strings.Replace(gvk.Group, ".", "-", -1) + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind), nil
}

// isAlreadyHandled method check if there is already an admission webhook registered for a given path
func isAlreadyHandled(path string, mgr manager.Manager) bool {
	if mgr.GetWebhookServer().WebhookMux == nil {
		return false
	}
	h, p := mgr.GetWebhookServer().WebhookMux.Handler(&http.Request{URL: &url.URL{Path: path}})
	if p == path && h != nil {
		return true
	}
	return false
}
