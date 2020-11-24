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
	"path/filepath"
	"strings"
	"time"

	apiext1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery/cached/memory"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kudobuilder/kudo/pkg/apis"
	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/operator"
	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"
	"github.com/kudobuilder/kudo/pkg/kubernetes"
	"github.com/kudobuilder/kudo/pkg/version"
	kudohook "github.com/kudobuilder/kudo/pkg/webhook"
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

func getEnv(key, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		val = def
	}
	return val
}

func main() {
	// Get version of KUDO
	log.Printf("KUDO Version: %#v", version.Get())

	// create new controller-runtime manager
	syncPeriod, err := parseSyncPeriod()
	if err != nil {
		log.Printf("Unable to parse manager sync period variable: %v", err)
		os.Exit(1)
	}

	if syncPeriod != nil {
		log.Print(fmt.Sprintf("Setting up manager, sync-period is %v:", syncPeriod))
	} else {
		log.Print("Setting up manager")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		CertDir:    getEnv("KUDO_CERT_DIR", filepath.Join("/tmp", "cert")),
		SyncPeriod: syncPeriod,
	})
	if err != nil {
		log.Printf("Unable to start manager: %v", err)
		os.Exit(1)
	}

	log.Print("Registering Components")

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("Unable to add APIs to scheme: %v", err)
	}
	log.Print("Scheme initialization")

	// We want both extensionapis registered so we can handle them correctly and typed, for example in the health util
	if err := apiext1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("Unable to add extension APIs v1 to scheme: %v", err)
	}
	if err := apiextv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("Unable to add extension APIs v1beta1 to scheme: %v", err)
	}

	// Setup all Controllers
	err = (&operator.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("Unable to register operator controller to the manager: %v", err)
		os.Exit(1)
	}
	log.Print("Operator controller set up")

	err = (&operatorversion.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("Unable to register operator controller to the manager: %v", err)
		os.Exit(1)
	}
	log.Print("OperatorVersion controller set up")

	discoveryClient, err := kubernetes.GetDiscoveryClient(mgr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)

	err = (&instance.Reconciler{
		Client:    mgr.GetClient(),
		Config:    mgr.GetConfig(),
		Discovery: cachedDiscoveryClient,
		Recorder:  mgr.GetEventRecorderFor("instance-controller"),
		Scheme:    mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Printf("Unable to register instance controller to the manager: %v", err)
		os.Exit(1)
	}
	log.Print("Instance controller set up")

	log.Printf("Setting up admission webhook")

	iac, err := kudohook.NewInstanceAdmission(mgr.GetConfig(), mgr.GetScheme())
	if err != nil {
		log.Printf("Unable to create an uncached client for the webhook: %v", err)
		os.Exit(1)
	}

	if err := registerWebhook("/admit", &kudoapi.Instance{}, &webhook.Admission{Handler: iac}, mgr); err != nil {
		log.Printf("Unable to create instance admission webhook: %v", err)
		os.Exit(1)
	}
	log.Printf("Instance admission webhook set up")

	// Add more webhooks below using the above registerWebhook method

	// Start the KUDO manager
	log.Print("Done! Everything is setup, starting KUDO manager now")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Printf("Unable to run the manager: %v", err)
		os.Exit(1)
	}
}

// registerWebhook method registers passed webhook using a give prefix (e.g. "/validate") and runtime object
// (e.g. kudoapi.Instance) to generate a webhook path e.g. "/validate-kudo-dev-v1beta1-instances". Webhook
// has to implement http.Handler interface (see kudoapi.InstanceAdmission for an example)
//
// NOTE: generated webhook path HAS to match the one used in the webhook configuration. See for example how
// MutatingWebhookConfiguration.Webhooks[0].ClientConfig.Service.Path is set in
// pkg/kudoctl/kudoinit/prereq/webhook.go::instanceAdmissionWebhook method
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
	return prefix + "-" + strings.ReplaceAll(gvk.Group, ".", "-") + "-" +
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
