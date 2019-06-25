// +build integration

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

package instance

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/onsi/gomega"

	"github.com/kudobuilder/kudo/pkg/apis"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var cfg *rest.Config

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crds")},
	}
	apis.AddToScheme(scheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}

// SetupTestReconcile returns a reconcile.Reconcile implementation that delegates to inner and
// writes the request to requests after Reconcile is finished.
func SetupTestReconcile(inner reconcile.Reconciler) (reconcile.Reconciler, chan reconcile.Request) {
	requests := make(chan reconcile.Request)
	fn := reconcile.Func(func(req reconcile.Request) (reconcile.Result, error) {
		result, err := inner.Reconcile(req)
		requests <- req
		return result, err
	})
	return fn, requests
}

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
		wg.Done()
	}()
	return stop, wg
}

func TestSpecParameterDifference(t *testing.T) {

	var testParams = []struct {
		name string
		new  map[string]string
		diff map[string]string
	}{
		{"update one value", map[string]string{"one": "11", "two": "2"}, map[string]string{"one": "11"}},
		{"update multiple values", map[string]string{"one": "11", "two": "22"}, map[string]string{"one": "11", "two": "22"}},
		{"add new value", map[string]string{"one": "1", "two": "2", "three": "3"}, map[string]string{"three": "3"}},
		{"remove one value", map[string]string{"one": "1"}, map[string]string{"two": "2"}},
		{"no difference", map[string]string{"one": "1", "two": "2"}, map[string]string{}},
		{"empty new map", map[string]string{}, map[string]string{"one": "1", "two": "2"}},
	}

	g := gomega.NewGomegaWithT(t)

	var old = map[string]string{"one": "1", "two": "2"}

	for _, test := range testParams {
		diff := parameterDifference(old, test.new)
		g.Expect(diff).Should(gomega.Equal(test.diff), test.name)
	}
}
