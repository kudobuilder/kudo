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

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/martian/log"
	"github.com/kudobuilder/kudo/pkg/apis"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/operator"
	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"
	util "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/kudobuilder/kudo/pkg/version"
	apiextenstionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	logf.SetLogger(zap.Logger(false))
	log := logf.Log.WithName("entrypoint")

	// Get version of KUDO
	log.Info(fmt.Sprintf("KUDO Version: %s", fmt.Sprintf("%#v", version.Get())))

	// create new controller-runtime manager
	log.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MapperProvider: util.NewDynamicRESTMapper,
		CertDir:        "/tmp/cert",
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

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		err = registerValidatingWebhook(&v1beta1.Instance{}, mgr)
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
func registerValidatingWebhook(obj runtime.Object, mgr manager.Manager) error {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return err
	}
	validator, isValidator := obj.(kudo.Validator)
	if !isValidator {
		log.Infof("skip registering a validating webhook, admission.Validator interface is not implemented", "GVK", gvk)
		return nil
	}
	vwh := kudo.ValidatingWebhookFor(validator)
	if vwh != nil {
		path := generateValidatePath(gvk)

		// Checking if the path is already registered.
		// If so, just skip it.
		if !isAlreadyHandled(path, mgr) {
			log.Infof("Registering a validating webhook",
				"GVK", gvk,
				"path", path)
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
