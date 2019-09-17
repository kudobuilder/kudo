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

package controller

import (
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/planexecution"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddControllerToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddControllerToManagerFuncs = []func(manager.Manager) error{
	planexecution.Add,
	instance.Add,
}

// AddControllersToManager adds all Controllers to the Manager
func AddControllersToManager(m manager.Manager) error {
	for _, f := range AddControllerToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
