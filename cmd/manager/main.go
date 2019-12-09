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
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apiextenstionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kudobuilder/kudo/pkg/apis"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/operator"
	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
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
	logf.SetLogger(zap.New(zap.UseDevMode(false)))
	log := logf.Log.WithName("entrypoint")

	// Get version of KUDO
	log.Info(fmt.Sprintf("KUDO Version: %s", fmt.Sprintf("%#v", version.Get())))

	// create new controller-runtime manager
	syncPeriod, err := parseSyncPeriod()
	if err != nil {
		log.Error(err, "unable to parse manager sync period variable")
		os.Exit(1)
	}

	if syncPeriod != nil {
		log.Info(fmt.Sprintf("setting up manager, sync-period is %v", syncPeriod))
	} else {
		log.Info(fmt.Sprintf("setting up manager"))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		CertDir:    "/tmp/cert",
		SyncPeriod: syncPeriod,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	log.Info("setting up scheme")

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable add APIs to scheme")
	}

	if err := apiextenstionsv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable to add extension APIs to scheme")
	}

	// Setup all Controllers

	log.Info("Setting up operator controller")
	err = (&operator.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Error(err, "unable to register operator controller to the manager")
		os.Exit(1)
	}

	log.Info("Setting up operator version controller")
	err = (&operatorversion.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Error(err, "unable to register operator controller to the manager")
		os.Exit(1)
	}

	log.Info("Setting up instance controller")
	err = (&instance.Reconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("instance-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Error(err, "unable to register instance controller to the manager")
		os.Exit(1)
	}

	if strings.ToLower(os.Getenv("ENABLE_WEBHOOKS")) == "true" {
		err = registerValidatingWebhook(&v1beta1.Instance{}, mgr, log)
		if err != nil {
			log.Error(err, "unable to create webhook")
			os.Exit(1)
		}
	}

	// Start the Cmd
	log.Info("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}

// this is a fork of a code in controller-runtime to be able to pass in our own Validator interface
// see kudo.Validator docs for more details\
//
// ideally in the future we should switch to just simply doing
// err = ctrl.NewWebhookManagedBy(mgr).
// For(&v1beta1.Instance{}).
// Complete()
//
// that internally calls this method but using their own internal Validator type
func registerValidatingWebhook(obj runtime.Object, mgr manager.Manager, log logr.Logger) error {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return err
	}
	validator, ok := obj.(kudo.Validator)
	if !ok {
		log.Info("skip registering a validating webhook, kudo.Validator interface is not implemented %v", gvk)

		return nil
	}
	vwh := kudo.ValidatingWebhookFor(validator)
	if vwh != nil {
		path := generateValidatePath(gvk)

		// Checking if the path is already registered.
		// If so, just skip it.
		if !isAlreadyHandled(path, mgr) {
			log.Info("Registering a validating webhook for %v on path %s", gvk, path)
			mgr.GetWebhookServer().Register(path, vwh)
		}
	}
	return nil
}

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

// if the strategy to generate this path changes we should update init code and webhook setup
// right now this is in sync how controller-runtime generates these paths
func generateValidatePath(gvk schema.GroupVersionKind) string {
	return "/validate-" + strings.Replace(gvk.Group, ".", "-", -1) + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind)
}
