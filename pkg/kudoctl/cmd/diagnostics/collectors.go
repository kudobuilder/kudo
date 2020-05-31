package diagnostics

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task/podexec"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	kudoutil "github.com/kudobuilder/kudo/pkg/util/kudo"
)

// resourceCollector - collector interface implementation for Kubernetes resources (runtime objects)
type resourceCollector struct {
	loadResourceFn func() (runtime.Object, error)
	name           string               // object kind used to describe the error
	parentDir      func() string        // parent dir to attach the printer's output
	failOnError    bool                 // define whether the collector should return the error
	callback       func(runtime.Object) // will be called with the retrieved resource after collection to update shared context
	printer        *nonFailingPrinter
	printMode      printMode
}

// collect - load a resource and send either the resource or collection error to printer
// return error if failOnError field is set to true
// if failOnError is true, finding no object(s) is treated as an error
func (c *resourceCollector) collect() error {
	obj, err := c._collect(c.failOnError)
	if err != nil {
		c.printer.printError(err, c.parentDir(), c.name)
		if c.failOnError {
			return err
		}
	}
	if obj != nil {
		c.printer.printObject(obj, c.parentDir(), c.printMode)
	}
	return nil
}

func (c *resourceCollector) _collect(failOnError bool) (runtime.Object, error) {
	obj, err := c.loadResourceFn()
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to retrieve object(s) of kind %s: %v", c.name, err)
	case obj == nil || reflect.ValueOf(obj).IsNil() || meta.IsListType(obj) && meta.LenList(obj) == 0:
		obj = nil
		if failOnError {
			return nil, fmt.Errorf("no object(s) of kind %s retrieved", c.name)
		}
	default:
		if c.callback != nil {
			c.callback(obj)
		}
	}
	return obj, nil
}

// resourceCollectorGroup - a composite collector for Kubernetes runtime objects whose loading and printing depend on
// each other's side-effects on the shared context
type resourceCollectorGroup struct {
	collectors []resourceCollector
	parentDir  func() string
}

// collect - collect resource and run callback for each collector, print all afterwards
// collection failures are treated as fatal regardless of the collectors failOnError flag setting
func (g resourceCollectorGroup) collect() error {
	objs := make([]runtime.Object, len(g.collectors))
	modes := make([]printMode, len(g.collectors))
	for i, c := range g.collectors {
		obj, err := c._collect(true)
		if err != nil {
			c.printer.printError(err, g.parentDir(), c.name)
			return err
		}
		objs[i], modes[i] = obj, c.printMode
	}
	for i, c := range g.collectors {
		c.printer.printObject(objs[i], c.parentDir(), modes[i])
	}
	return nil
}

type logsCollector struct {
	loadLogFn func(string, string) (io.ReadCloser, error)
	pods      []v1.Pod
	parentDir func() string
	printer   *nonFailingPrinter
}

// TODO: consider storing logs with commands and file copies: filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), container.Name)
func (c *logsCollector) collect() error {
	for _, pod := range c.pods {
		for _, container := range pod.Spec.Containers {
			log, err := c.loadLogFn(pod.Name, container.Name)
			if err != nil {
				c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), fmt.Sprintf("%s.log", container.Name))
			} else {
				c.printer.printLog(log, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name)), container.Name)
				_ = log.Close()
			}
		}
	}
	return nil
}

type dependencyCollector struct {
	s         *env.Settings
	ir        *resourceFuncsConfig
	parentDir stringGetter
	printer   *nonFailingPrinter
}

func (c *dependencyCollector) collect() error {
	instances, err := c.ir.instanceDependencies()
	if err != nil {
		err = fmt.Errorf("failed to retrieve dependencies for %s: %v", c.ir.instanceObj.Name, err)
	}
	// TODO: disabled as at this moment it's not known how to distinguish between having no deps or failure to find them
	//else if instances == nil || len(instances) == 0 {
	//	err = fmt.Errorf("no dependencies for %s retrieved", c.ir.instanceObj.Name)
	//}
	if err != nil {
		c.printer.printError(err, c.parentDir(), "dependencies")
		return nil // discard the error because it should not be fatal for the parent instance
	}

	for i := range instances {
		instance := &instances[i]
		ir := &resourceFuncsConfig{
			c:           c.ir.c,
			ns:          c.ir.ns,
			instanceObj: instance,
			opts:        metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", kudoutil.OperatorLabel, instance.Labels[kudoutil.OperatorLabel])},
			logOpts:     c.ir.logOpts,
		}
		_ = runForInstance(c.s, ir, &processingContext{root: c.parentDir(), instanceName: instance.Name}, c.printer)
		// ignore runner.fatalErr as it must have been printed by the collector who threw it
	}
	return nil
}

func _containsMap(m1, m2 map[string]string) bool {
	for k, v := range m2 {
		if m1[k] != v {
			return false
		}
	}
	return true
}

type fileCollector struct {
	s         *env.Settings
	pods      []v1.Pod
	copySpecs []v1beta1.DiagnosticResourceSpec
	parentDir stringGetter
	printer   *nonFailingPrinter
}

func (c *fileCollector) collect() error {
	restCfg, _ := kube.GetConfig(c.s.KubeConfig).ClientConfig() // TODO: err -> print and quit
	for _, copySpec := range c.copySpecs {
		// filter pods by selector
		var pods []v1.Pod
		for _, pod := range c.pods {
			if _containsMap(pod.Labels, copySpec.Selectors.MatchLabels) {
				pods = append(pods, pod)
			}
		}
		// for each filtered pod run copy commands for the corresponding containers
		for _, pod := range pods {
			containers := map[string]struct{}{}
			for _, container := range pod.Spec.Containers {
				containers[container.Name] = struct{}{}
			}
			for _, copyCmd := range copySpec.CopyCmds {
				stdout := bytes.Buffer{}
				stderr := strings.Builder{}

				pe := &podexec.PodExec{
					RestCfg:       restCfg,
					PodName:       pod.Name,
					PodNamespace:  pod.Namespace,
					ContainerName: copyCmd.Container,
					Args:          []string{"cat", copyCmd.Source},
					In:            nil,
					Out:           &stdout,
					Err:           &stderr,
				}
				if err := pe.Run(); err != nil {
					err := fmt.Errorf("%wfailed to copy file. err: %v, stderr: %s, stdout: %s", podexec.ErrCommandFailed, err, stderr.String(), stdout.String())
					c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), copyCmd.Container), copyCmd.Dest)
					continue
				}
				c.printer.printStream(&stdout, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), copyCmd.Container), copyCmd.Dest)
			}
		}

	}

	return nil
}

type commandCollector struct {
	s            *env.Settings
	pods         []v1.Pod
	commandSpecs []v1beta1.DiagnosticResourceSpec
	parentDir    stringGetter
	printer      *nonFailingPrinter
}

func (c *commandCollector) collect() error {
	restCfg, _ := kube.GetConfig(c.s.KubeConfig).ClientConfig() // TODO: err -> print and quit
	for _, commandSpec := range c.commandSpecs {
		// filter pods by selector
		var pods []v1.Pod
		for _, pod := range c.pods {
			if _containsMap(pod.Labels, commandSpec.Selectors.MatchLabels) {
				pods = append(pods, pod)
			}
		}
		// for each filtered pod run copy commands for the corresponding containers
		for _, pod := range pods {
			containers := map[string]struct{}{}
			for _, container := range pod.Spec.Containers {
				containers[container.Name] = struct{}{}
			}
			for _, cmd := range commandSpec.Commands {
				if _, ok := containers[cmd.Container]; !ok {
					c.printer.printError(
						fmt.Errorf("container %s not found for pod %s", cmd.Container, pod.Name),
						filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), cmd.Container),
						fmt.Sprintf("%s.err", cmd.Output))
					continue
				}

				stdout := bytes.Buffer{}
				stderr := strings.Builder{}

				pe := &podexec.PodExec{
					RestCfg:       restCfg,
					PodName:       pod.Name,
					PodNamespace:  pod.Namespace,
					ContainerName: cmd.Container,
					Args:          []string{cmd.Exec},
					In:            nil,
					Out:           &stdout,
					Err:           &stderr,
				}
				if err := pe.Run(); err != nil {
					err = fmt.Errorf("%wfailed to execute '%s'. err: %v, stderr: %s, stdout: %s", podexec.ErrCommandFailed, cmd.Exec, err, stderr.String(), stdout.String())
					c.printer.printError(err, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), cmd.Container), cmd.Output)
					continue
				}
				c.printer.printStream(&stdout, filepath.Join(c.parentDir(), fmt.Sprintf("pod_%s", pod.Name), cmd.Container), cmd.Output)
			}
		}

	}

	return nil
}
