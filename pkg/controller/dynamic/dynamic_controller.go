package dynamic

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
)

type ControllerRegistry struct {
	controllers map[string](chan struct{})
	errChan     chan error
	sync.Mutex
	mgr manager.Manager
}

func (c *ControllerRegistry) Register(u unstructured.Unstructured) error {
	c.Lock()
	defer c.Unlock()
	if c.controllers == nil {
		c.controllers = map[string](chan struct{}){}
	}
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	stopMgr := make(chan struct{})
	if c.mgr == nil {
		mgr, err := manager.New(cfg, manager.Options{})
		c.mgr = mgr
		if err != nil {
			return err
		}

		go func() {
			c.mgr.Start(stopMgr)
			fmt.Println("STOPPING MANAGER OH NO")
		}()
	}

	ctrlName := fmt.Sprintf("%s-%s", u.GroupVersionKind().String(), "controller")
	ctrl, err := controller.New(ctrlName, c.mgr, controller.Options{
		Reconciler: reconcile.Func(func(o reconcile.Request) (reconcile.Result, error) {
			fmt.Println("RECONCILING DYNAMIC THING")
			return reconcile.Result{}, nil
		}),
	})

	if err := ctrl.Watch(&source.Kind{Type: &u}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	stopCh := make(chan struct{})
	go ctrl.Start(stopCh)
	c.controllers[u.GroupVersionKind().String()] = stopCh

	return nil
}

func (c *ControllerRegistry) Stop(u unstructured.Unstructured) error {
	c.Lock()
	gvk := u.GroupVersionKind().String()
	if stopCh, ok := c.controllers[gvk]; ok {
		close(stopCh)
	}
	delete(c.controllers, gvk)
	c.Unlock()
	return nil
}
