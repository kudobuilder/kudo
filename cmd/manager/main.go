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
	"os"

	"github.com/kudobuilder/kudo/pkg/controller/instance"

	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"

	"github.com/kudobuilder/kudo/pkg/controller/operator"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/version"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	kudov1alpha1.AddToScheme(scheme)
}

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("entrypoint")

	// Get version of KUDO
	log.Info(fmt.Sprintf("KUDO Version: %s", fmt.Sprintf("%#v", version.Get())))

	// create new controller-runtime manager
	log.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

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

	// Start the Cmd
	log.Info("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}
