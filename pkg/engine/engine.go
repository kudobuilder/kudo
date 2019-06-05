package engine

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"text/template"
	"time"

	"github.com/kudobuilder/kudo/pkg/util/health"
	ktemplate "github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/masterminds/sprig"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Engine is the control struct for parsing and templating Kubernetes resources in an ordered fashion
type Engine struct {
	FuncMap template.FuncMap
}

// New creates an engine with a default function map, using a modified Sprig func map. Because these
// templates are rendered by the operator, we delete any functions that potentially access the environment
// the controller is running in.
func New() *Engine {
	f := sprig.TxtFuncMap()

	// Prevent environment access inside the running KUDO Controller
	funcs := []string{"env", "expandenv", "base", "dir", "clean", "ext", "isAbs"}

	for _, fun := range funcs {
		delete(f, fun)
	}

	return &Engine{
		FuncMap: f,
	}
}

// Render creates a fully rendered template based on a set of values. It parses these in strict mode,
// returning errors when keys are missing.
func (e *Engine) Render(tpl string, vals map[string]interface{}) (string, error) {
	t := template.New("gotpl")
	t.Option("missingkey=error")

	var buf bytes.Buffer
	t = t.New("tpl").Funcs(e.FuncMap)

	if _, err := t.Parse(tpl); err != nil {
		return "", fmt.Errorf("error parsing template: %s", err)
	}

	if err := t.ExecuteTemplate(&buf, "tpl", vals); err != nil {
		return "", fmt.Errorf("error rendering template: %s", err)
	}

	return buf.String(), nil
}

// Watch for Deployments, Jobs and StatefulSets
// Define a mapping from the object in the event to one or more
// objects to Reconcile.  Specifically this calls for
// a reconciliation of any objects "Owner".
func ReconcileRequestsMapperFunc(mgr manager.Manager) handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
		owners := a.Meta.GetOwnerReferences()
		requests := make([]reconcile.Request, 0)
		for _, owner := range owners {
			// if owner is an instance, we also want to queue up the
			// PlanExecution in the Status section
			inst := &kudov1alpha1.Instance{}
			err := mgr.GetClient().Get(context.TODO(), client.ObjectKey{
				Name:      owner.Name,
				Namespace: a.Meta.GetNamespace(),
			}, inst)

			if err != nil {
				log.Printf("Error getting instance object: %v", err)
			} else {
				log.Printf("Adding \"%v\" to reconcile", inst.Status.ActivePlan.Name)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      inst.Status.ActivePlan.Name,
						Namespace: inst.Status.ActivePlan.Namespace,
					},
				})
			}
		}
		return requests
	}
}

func PlanEventPredicateFunc() predicate.Funcs {
	// 'UpdateFunc' and 'CreateFunc' used to judge if a event about the object is
	// what we want. If that is true, the event will be processed by the reconciler.
	// PlanExecutions should be mostly immutable.  Updates should only
	msg := "PlanEventPredicate: Received update event for an instance named: %v"
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.Printf(msg, e.MetaNew.GetName())
			return e.ObjectOld != e.ObjectNew
		},
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf(msg, e.Meta.GetName())
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// TODO send event for Instance that plan was deleted
			log.Printf(msg, e.Meta.GetName())
			return true
		},
	}
}

func InstanceEventPredicateFunc(mgr manager.Manager) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			old := e.ObjectOld.(*kudov1alpha1.Instance)
			new := e.ObjectNew.(*kudov1alpha1.Instance)

			//Haven't done anything yet
			if new.Status.ActivePlan.Name == "" {
				err := CreatePlan(mgr, "deploy", new)
				if err != nil {
					log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", "deploy", new.Name, err)
				}
				return true
			}

			//get the new FrameworkVersion object
			fv := &kudov1alpha1.FrameworkVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      new.Spec.FrameworkVersion.Name,
					Namespace: new.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error getting frameworkversion \"%v\" for instance \"%v\": %v",
					new.Spec.FrameworkVersion.Name,
					new.Name,
					err)
				//TODO
				//We probably want to handle this differently and mark this instance as unhealthy
				//since its linking to a bad FV
				return false
			}
			//Identify plan to be executed by this change
			var planName string
			var ok bool
			if old.Spec.FrameworkVersion != new.Spec.FrameworkVersion {
				//Its an Upgrade!
				_, ok = fv.Spec.Plans["upgrade"]
				if !ok {
					_, ok = fv.Spec.Plans["update"]
					if !ok {
						_, ok = fv.Spec.Plans["deploy"]
						if !ok {
							log.Println("InstanceEventPredicate: Could not find any plan to use for upgrade")
							return false
						}
						ok = true // Do we need this here?
						planName = "deploy"
					} else {
						planName = "update"
					}
				} else {
					planName = "upgrade"
				}
			} else if !reflect.DeepEqual(old.Spec, new.Spec) {
				for k, v := range new.Spec.Parameters {
					if old.Spec.Parameters[k] != v {
						//Find the right parameter in the FV
						for _, param := range fv.Spec.Parameters {
							if param.Name == k {
								planName = param.Trigger
								ok = true
							}
						}
						if !ok {
							log.Printf("InstanceEventPredicate: Instance %v updated parameter %v, but parameter not found in FrameworkVersion %v\n", new.Name, k, fv.Name)
						} else if planName == "" {
							_, ok = fv.Spec.Plans["update"]
							if !ok {
								_, ok = fv.Spec.Plans["deploy"]
								if !ok {
									log.Println("InstanceEventPredicate: Could not find any plan to use for update")
								} else {
									planName = "deploy"
								}
							} else {
								planName = "update"
							}
							log.Printf("InstanceEventPredicate: Instance %v updated parameter %v, but no specified trigger.  Using default plan %v\n", new.Name, k, planName)
						}
					}
				}
				//Not currently doing anything for Dependency changes
			} else {
				log.Println("InstanceEventPredicate: Old and new spec matched...")
				planName = "deploy"
			}
			log.Printf("InstanceEventPredicate: Going to call plan \"%v\"", planName)

			//we found something
			if ok {

				//mark the current plan as Suspend,
				current := &kudov1alpha1.PlanExecution{}
				err = mgr.GetClient().Get(context.TODO(), client.ObjectKey{Name: new.Status.ActivePlan.Name, Namespace: new.Status.ActivePlan.Namespace}, current)
				if err != nil {
					log.Printf("InstanceEventPredicate: Ignoring error when getting plan for new instance: %v", err)
				} else {
					if current.Status.State == kudov1alpha1.PhaseStateComplete {
						log.Println("InstanceEventPredicate: Current Plan for Instance is already done, won't change the Suspend flag.")
					} else {
						log.Println("InstanceEventPredicate: Setting PlanExecution to Suspend")
						t := true
						current.Spec.Suspend = &t
						did, err := controllerutil.CreateOrUpdate(context.TODO(), mgr.GetClient(), current, func(o runtime.Object) error {
							t := true
							o.(*kudov1alpha1.PlanExecution).Spec.Suspend = &t
							return nil
						})
						if err != nil {
							log.Printf("InstanceEventPredicate: Error changing the current PlanExecution to Suspend: %v", err)
						} else {
							log.Printf("InstanceEventPredicate: No error in setting PlanExecution.Suspend to true. Returned %v", did)
						}
					}
				}

				err = CreatePlan(mgr, planName, new)
				if err != nil {
					log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", planName, new.Name, err)
				}
			}

			//status change?  Sent it along

			//See if there's a current plan being run.
			//if so "cancel" the plan run
			//create a new plan
			return e.ObjectOld != e.ObjectNew
		},
		//New Instances should have Deploy called
		CreateFunc: func(e event.CreateEvent) bool {
			log.Printf("InstanceEventPredicate: Received create event for an instance named: %v", e.Meta.GetName())
			instance := e.Object.(*kudov1alpha1.Instance)

			//get the instance FrameworkVersion object
			fv := &kudov1alpha1.FrameworkVersion{}
			err := mgr.GetClient().Get(context.TODO(),
				types.NamespacedName{
					Name:      instance.Spec.FrameworkVersion.Name,
					Namespace: instance.Spec.FrameworkVersion.Namespace,
				},
				fv)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error getting frameworkversion \"%v\" for instance \"%v\": %v",
					instance.Spec.FrameworkVersion.Name,
					instance.Name,
					err)
				//TODO
				//We probably want to handle this differently and mark this instance as unhealthy
				//since its linking to a bad FV
				return false
			}
			planName := "deploy"

			if _, ok := fv.Spec.Plans[planName]; !ok {
				log.Println("InstanceEventPredicate: Could not find deploy plan")
				return false
			}

			err = CreatePlan(mgr, planName, instance)
			if err != nil {
				log.Printf("InstanceEventPredicate: Error creating \"%v\" object for \"%v\": %v", planName, instance.Name, err)
			}
			return err == nil
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.Printf("InstanceEventPredicate: Received delete event for an instance named: %v", e.Meta.GetName())
			return true
		},
	}
}

func CreatePlan(mgr manager.Manager, planName string, instance *kudov1alpha1.Instance) error {
	gvk, _ := apiutil.GVKForObject(instance, mgr.GetScheme())
	recorder := mgr.GetRecorder("instance-controller")
	recorder.Event(instance, "Normal", "CreatePlanExecution", fmt.Sprintf("Creating \"%v\" plan execution", planName))

	// Create a new ref
	ref := corev1.ObjectReference{
		Kind:      gvk.Kind,
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}

	planExecution := kudov1alpha1.PlanExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v-%v", instance.Name, planName, time.Now().Nanosecond()),
			Namespace: instance.GetNamespace(),
			//Should also add one for Framework in here as well.
			Labels: map[string]string{
				"framework-version": instance.Spec.FrameworkVersion.Name,
				"instance":          instance.Name,
			},
		},
		Spec: kudov1alpha1.PlanExecutionSpec{
			Instance: ref,
			PlanName: planName,
		},
	}
	//Make this instance the owner of the PlanExecution
	if err := controllerutil.SetControllerReference(instance, &planExecution, mgr.GetScheme()); err != nil {
		log.Printf("InstanceController: Error setting ControllerReference")
		return err
	}
	//new!
	if err := mgr.GetClient().Create(context.TODO(), &planExecution); err != nil {
		log.Printf("InstanceController: Error creating planexecution \"%v\": %v", planExecution.Name, err)
		recorder.Event(instance, "Warning", "CreatePlanExecution", fmt.Sprintf("Error creating planexecution \"%v\": %v", planExecution.Name, err))
		return err
	}
	recorder.Event(instance, "Normal", "PlanCreated", fmt.Sprintf("PlanExecution \"%v\" created", planExecution.Name))
	return nil
}

func PopulatePlanExecutionPhases(basePath string, executedPlan *kudov1alpha1.Plan, planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, frameworkVersion *kudov1alpha1.FrameworkVersion, configs map[string]interface{}, recorder record.EventRecorder) error {
	planExecution.Status.Phases = make([]kudov1alpha1.PhaseStatus, len(executedPlan.Phases))
	var err error
	for i, phase := range executedPlan.Phases {
		planExecution.Status.Phases[i].Name = phase.Name
		planExecution.Status.Phases[i].Strategy = phase.Strategy
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStatePending
		planExecution.Status.Phases[i].Steps = make([]kudov1alpha1.StepStatus, len(phase.Steps))
		for j, step := range phase.Steps {
			// fetch FrameworkVersion
			// get the task name from the step
			// get the task definition from the FV
			// create the kustomize templates
			// apply
			configs["PlanName"] = planExecution.Spec.PlanName
			configs["PhaseName"] = phase.Name
			configs["StepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)

			var objs []runtime.Object
			engine := New()

			for _, t := range step.Tasks {
				// resolve task
				if taskSpec, ok := frameworkVersion.Spec.Tasks[t]; ok {
					var resources []string
					fsys := fs.MakeFakeFS()

					for _, res := range taskSpec.Resources {
						if resource, ok := frameworkVersion.Spec.Templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error expanding template: %v", err))
								log.Printf("PlanExecutionController: Error expanding template: %v", err)
							}
							fsys.WriteFile(fmt.Sprintf("%s/%s", basePath, res), []byte(templatedYaml))
							resources = append(resources, res)

						} else {
							recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding resource named %v for framework version %v", res, frameworkVersion.Name))
							log.Printf("PlanExecutionController: Error finding resource named %v for framework version %v", res, frameworkVersion.Name)
							return err
						}
					}

					kustomization := &ktypes.Kustomization{
						NamePrefix: instance.Name + "-",
						Namespace:  instance.Namespace,
						CommonLabels: map[string]string{
							"heritage":      "kudo",
							"app":           frameworkVersion.Spec.Framework.Name,
							"version":       frameworkVersion.Spec.Version,
							"instance":      instance.Name,
							"planexecution": planExecution.Name,
							"plan":          planExecution.Spec.PlanName,
							"phase":         phase.Name,
							"step":          step.Name,
						},
						GeneratorOptions: &ktypes.GeneratorOptions{
							DisableNameSuffixHash: true,
						},
						Resources:             resources,
						PatchesStrategicMerge: []patch.StrategicMerge{},
					}

					yamlBytes, err := yaml.Marshal(kustomization)
					if err != nil {
						return err
					}

					fsys.WriteFile(fmt.Sprintf("%s/kustomization.yaml", basePath), yamlBytes)

					ldr, err := loader.NewLoader(basePath, fsys)
					if err != nil {
						return err
					}
					defer ldr.Cleanup()

					rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()))
					kt, err := target.NewKustTarget(ldr, fsys, rf, transformer.NewFactoryImpl())
					if err != nil {
						return err
					}

					allResources, err := kt.MakeCustomizedResMap()
					if err != nil {
						return err
					}

					res, err := allResources.EncodeAsYaml()
					if err != nil {
						return err
					}

					objsToAdd, err := ktemplate.ParseKubernetesObjects(string(res))
					if err != nil {
						recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err))
						log.Printf("PlanExecutionController: Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecution.Name, err)
						return err
					}
					objs = append(objs, objsToAdd...)
				} else {
					recorder.Event(planExecution, "Warning", "InvalidPlanExecution", fmt.Sprintf("Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name))
					log.Printf("PlanExecutionController: Error finding task named %s for framework version %s", taskSpec, frameworkVersion.Name)
					return nil
				}
			}

			planExecution.Status.Phases[i].Steps[j].Name = step.Name
			planExecution.Status.Phases[i].Steps[j].Objects = objs
			planExecution.Status.Phases[i].Steps[j].Delete = step.Delete
			log.Printf("PlanExecutionController: Phase \"%v\" Step \"%v\" has %v object(s)", phase.Name, step.Name, len(objs))
		}
	}
	return nil
}

func MutateFn(oldObj runtime.Object) controllerutil.MutateFn {
	return func(newObj runtime.Object) error {
		//TODO Clean this up.  I don't like having to do a switch here
		switch t := newObj.(type) {
		case *appsv1.StatefulSet:
			log.Printf("PlanExecutionController: CreateOrUpdate: StatefulSet %+v", t.Name)

			newSs := newObj.(*appsv1.StatefulSet)
			ss, ok := oldObj.(*appsv1.StatefulSet)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected StatefulSet type")
			}

			// We need some specialized logic in there.  We can't just copy the Spec since there are other values
			// like spec.updateState, spec.volumeClaimTemplates, etc that are all
			// generated from the object by the k8s controller.  We just want to update things we can change
			newSs.Spec.Replicas = ss.Spec.Replicas

			return nil
		case *appsv1.Deployment:
			newD := newObj.(*appsv1.Deployment)
			d, ok := oldObj.(*appsv1.Deployment)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected deployment type")
			}
			newD.Spec.Replicas = d.Spec.Replicas
			return nil
		case *v1beta1.Deployment:
			newD := newObj.(*v1beta1.Deployment)
			d, ok := oldObj.(*v1beta1.Deployment)
			if !ok {
				return fmt.Errorf("object passed in doesn't match expected deployment type")
			}
			newD.Spec.Replicas = d.Spec.Replicas
			return nil

		case *batchv1.Job:
			// job := oldObj.(*batchv1.Job)

		case *kudov1alpha1.Instance:
			// i := oldObj.(*kudov1alpha1.Instance)

		//unless we build logic for what a healthy object is, assume its healthy when created
		default:
			log.Print("PlanExecutionController: CreateOrUpdate: Type is not implemented yet")
			return nil
		}

		return nil

	}
}

func Cleanup(c client.Client, obj runtime.Object) error {
	switch obj := obj.(type) {
	case *batchv1.Job:
		//We need to see if there's a current job on the system that matches this exactly (with labels)
		log.Printf("PlanExecutionController.Cleanup: *batchv1.Job %v", obj.Name)

		present := &batchv1.Job{}
		key, _ := client.ObjectKeyFromObject(obj)
		err := c.Get(context.TODO(), key, present)
		if errors.IsNotFound(err) {
			//this is fine, its good to go
			log.Printf("PlanExecutionController: Could not find job \"%v\" in cluster. Good to make a new one.", key)
			return nil
		}
		if err != nil {
			//Something else happened
			return err
		}
		//see if the job in the cluster has the same labels as the one we're looking to add.
		for k, v := range obj.Labels {
			if v != present.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, present.Labels[k])
				err = c.Delete(context.TODO(), present)
				return err
			}
		}
		for k, v := range present.Labels {
			if v != obj.Labels[k] {
				//need to delete the present job since its got labels that aren't the same
				log.Printf("PlanExecutionController: Different values for job key \"%v\": \"%v\" and \"%v\"", k, v, obj.Labels[k])
				err = c.Delete(context.TODO(), present)
				return err
			}
		}
		return nil
	}

	return nil
}

func RunPhases(executedPlan *kudov1alpha1.Plan, planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, c client.Client, scheme *runtime.Scheme) error {
	var err error
	for i, phase := range planExecution.Status.Phases {
		//If we still want to execute phases in this plan
		//check if phase is healthy
		for j, s := range phase.Steps {
			planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateComplete

			for _, obj := range s.Objects {
				if s.Delete {
					log.Printf("PlanExecutionController: Step \"%v\" was marked to delete object %+v", s.Name, obj)
					err = c.Delete(context.TODO(), obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
					if errors.IsNotFound(err) || err == nil {
						//This is okay
						log.Printf("PlanExecutionController: Object was already deleted or did not exist in step \"%v\"", s.Name)
					}
					if err != nil {
						log.Printf("PlanExecutionController: Error deleting object in step \"%v\": %v", s.Name, err)
						planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
						planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError
						return err
					}
					continue
				}

				// Make sure this object is applied to the cluster. Get back the instance from
				// the cluster so we can see if it's healthy or not
				if err = controllerutil.SetControllerReference(instance, obj.(metav1.Object), scheme); err != nil {
					return err
				}

				//Some objects don't update well.  We capture the logic here to see if we need to cleanup the current object
				err = Cleanup(c, obj)
				if err != nil {
					log.Printf("PlanExecutionController: Cleanup failed: %v", err)
				}

				arg := obj.DeepCopyObject()
				result, err := controllerutil.CreateOrUpdate(context.TODO(), c, arg, MutateFn(obj))

				if err != nil {
					log.Printf("PlanExecutionController: Error CreateOrUpdate Object in step \"%v\": %v", s.Name, err)
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError

					return err
				}
				log.Printf("PlanExecutionController: CreateOrUpdate resulted in: %v", result)

				// get the existing object meta
				metaObj := obj.(metav1.Object)

				// retrieve the existing object
				key := client.ObjectKey{
					Name:      metaObj.GetName(),
					Namespace: metaObj.GetNamespace(),
				}

				err = c.Get(context.TODO(), key, obj)

				if err != nil {
					log.Printf("PlanExecutionController: Error getting new object in step \"%v\": %v", s.Name, err)
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateError
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateError
					return err
				}
				err = health.IsHealthy(c, obj)
				if err != nil {
					log.Printf("PlanExecutionController: Obj is NOT healthy: %+v", obj)
					planExecution.Status.Phases[i].Steps[j].State = kudov1alpha1.PhaseStateInProgress
					planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress
				}
			}
			log.Printf("PlanExecutionController: Phase \"%v\" has strategy %v", phase.Name, phase.Strategy)
			if phase.Strategy == kudov1alpha1.Serial {
				//we need to skip the rest of the steps if this step is unhealthy
				log.Printf("PlanExecutionController: Phase \"%v\" marked as serial", phase.Name)
				if planExecution.Status.Phases[i].Steps[j].State != kudov1alpha1.PhaseStateComplete {
					log.Printf("PlanExecutionController: Step \"%v\" isn't complete, skipping rest of steps in phase until it is", planExecution.Status.Phases[i].Steps[j].Name)
					break //break step loop
				} else {
					log.Printf("PlanExecutionController: Step \"%v\" is healthy, so I can continue on", planExecution.Status.Phases[i].Steps[j].Name)
				}
			}

			log.Printf("PlanExecutionController: Looked at step \"%v\"", s.Name)
		}
		if health.IsPhaseHealthy(planExecution.Status.Phases[i]) {
			log.Printf("PlanExecutionController: Phase \"%v\" marked as healthy", phase.Name)
			planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateComplete
			continue
		}

		//This phase isn't quite ready yet.  Lets see what needs to be done
		planExecution.Status.Phases[i].State = kudov1alpha1.PhaseStateInProgress

		//Don't keep going to other plans if we're flagged to perform the phases in serial
		if executedPlan.Strategy == kudov1alpha1.Serial {
			log.Printf("PlanExecutionController: Phase \"%v\" not healthy, and plan marked as serial, so breaking.", phase.Name)
			break
		}
		log.Printf("PlanExecutionController: Looked at phase \"%v\"", phase.Name)
	}
	return nil
}

func ParseConfig(planExecution *kudov1alpha1.PlanExecution, instance *kudov1alpha1.Instance, frameworkVersion *kudov1alpha1.FrameworkVersion, recorder record.EventRecorder) (map[string]interface{}, error) {
	//Load parameters:
	//Create config map to hold all parameters for instantiation
	configs := make(map[string]interface{})
	//Default parameters from instance metadata
	configs["FrameworkName"] = frameworkVersion.Spec.Framework.Name
	configs["Name"] = instance.Name
	configs["Namespace"] = instance.Namespace

	params := make(map[string]interface{})
	//parameters from instance spec
	for k, v := range instance.Spec.Parameters {
		if _, ok := configs[k]; ok {
			return nil, fmt.Errorf("cannot overwrite predefined config param %v with new value %v", k, v)
		}
		params[k] = v
	}
	//merge defaults with customizations
	for _, param := range frameworkVersion.Spec.Parameters {
		_, ok := params[param.Name]
		if !ok { //not specified in params
			if param.Required {
				err := fmt.Errorf("parameter %v was required but not provided by instance %v", param.Name, instance.Name)
				log.Printf("PlanExecutionController: %v", err)
				recorder.Event(planExecution, "Warning", "MissingParameter", fmt.Sprintf("Could not find required parameter (%v)", param.Name))
				return nil, err
			}
			params[param.Name] = param.Default
		}
	}
	configs["Params"] = params
	return configs, nil
}
