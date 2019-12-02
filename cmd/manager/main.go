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
	"time"
	"flag"

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

type args struct {

	// SyncPeriod determines the minimum frequency at which watched resources are
	// reconciled. A lower period will correct entropy more quickly, but reduce
	// responsiveness to change if there are many watched resources. Change this
	// value only if you know what you are doing. Defaults to 30 second if unset.
	syncPeriod time.Duration
}

func parseArgs() args {
	args := args{}

	flag.DurationVar(&args.syncPeriod, "sync-period", time.Duration(30)*time.Second,
		"SyncPeriod determines the minimum frequency at which watched resources are reconciled.")

	flag.Parse()
	return args
}

func main() {
	logf.SetLogger(zap.Logger(false))
	log := logf.Log.WithName("entrypoint")

	// Get version of KUDO
	log.Info(fmt.Sprintf("KUDO Version: %s", fmt.Sprintf("%#v", version.Get())))

	args := parseArgs()

	// create new controller-runtime manager
	log.Info(fmt.Sprintf("setting up manager, sync-period is %v", args.syncPeriod))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MapperProvider: util.NewDynamicRESTMapper,
		SyncPeriod: &args.syncPeriod,
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

	// Start the Cmd
	log.Info("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}
