package diagnostics

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/diagnostics/sonobuoy"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/version"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	nsKudoSystem      = "kudo-system"
	labelKudoOperator = "kudo.dev/operator"
	appKudoManager    = "kudo-manager"
)

var Options = struct {
	Instance string
}{}

type Collector interface {
	Collect(f writerFactory) error
}

func Collect(cmd *cobra.Command, settings *env.Settings) error {
	fmt.Println("Collecting diagnostics")

	config, _ := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	ns := settings.Namespace
	kc, _ := env.GetClient(settings)
	instance, _ := kc.GetInstance(Options.Instance, ns)
	c, _ := kube.GetKubeClient(settings.KubeConfig)

	// TODO: this is as terrible as it looks
	// run diagnostic resources
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		// TODO: OV obtained twice: here and in collectors
		o, _ := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, ns)
		args := pluginArgs(c, instance, o)
		pod, _ := createPluginPod(c, args)
		command := exec.Command("kubectl", "cp",
			fmt.Sprintf("%s/%s:/tmp/results/results.tar.gz", pod.Namespace, pod.Name), diagDir +"/results.tar.gz")
		if _, err := os.Stat(diagDir); os.IsNotExist(err) {
			err = os.MkdirAll(diagDir, 0700)
			if err != nil {
				log.Println(err)
			}
		}
		command.Stdout = os.Stdout
		time.Sleep(3 * time.Second)
		fmt.Println("executing ", command.String())
		err := command.Run()
		if err != nil {
			log.Println(err)
		}
		fmt.Println("deleting namespace ", pod.Namespace)
		c.KubeClient.CoreV1().Namespaces().Delete(pod.Namespace, &metav1.DeleteOptions{})
	}()


	byOperator := metav1.ListOptions{
		LabelSelector: labelKudoOperator + "=" + instance.Labels[labelKudoOperator],
	}
	byKudoManager := metav1.ListOptions{
		LabelSelector: "app=" + appKudoManager,
	}

	iResources := &resourceFuncs{c, kc, ns, byOperator, instance}
	cResources := &resourceFuncs{c, nil, nsKudoSystem, byKudoManager, nil} // no need for kudo and instance

	// TODO: use more meaningful variable names
	dc := resourceListCollector{getResources: iResources.deployments()}
	pc := resourceListCollector{getResources: iResources.pods()}
	ec := resourceListCollector{getResources: iResources.events()}
	sc := resourceListCollector{getResources: iResources.services()}
	rsc := resourceListCollector{getResources: iResources.replicaSets()}
	ovc := resourceCollector{getResource: iResources.operatorVersions()}
	oc := resourceCollector{getResource: iResources.operators(&ovc)}
	lc := logCollector{
		Client: c,
		ns:     ns,
		opts:   corev1.PodLogOptions{}, // TODO: add time range
		logs:   make(map[string]io.ReadCloser),
		pods:   &pc,
	}
	cpc := resourceListCollector{getResources: cResources.pods()}
	ssc := resourceListCollector{getResources: cResources.statefulSets()}
	rbc := resourceListCollector{getResources: cResources.roleBindings()}
	crbc := resourceListCollector{getResources: cResources.clusterRoleBindings()}
	rc := resourceListCollector{getResources: cResources.roles(&rbc)}
	crc := resourceListCollector{getResources: cResources.clusterRoles(&crbc)}
	clc := logCollector{
		Client: c,
		ns:     nsKudoSystem,
		opts:   corev1.PodLogOptions{}, // TODO: add time range
		logs:   make(map[string]io.ReadCloser),
		pods:   &cpc,
	}
	verc := dumpingCollector{s: version.Get()}
	setc := dumpingCollector{s: *settings}

	collectors := []Collector{&dc, &pc, &ec, &sc, &rsc, &ovc, &oc, &cpc, &ssc, &rbc, &crbc, &rc, &crc, &lc, &clc, &verc, &setc}
	var describers []Collector
	for _, c := range collectors {
		switch rc := c.(type) {
		case *resourceCollector:
			describers = append(describers, &describeCollector{config: config, d: rc})
		case *resourceListCollector:
			describers = append(describers, &describeListCollector{config: config, d: rc})
		}
	}
	collectors = append(collectors, describers...)

	var errors *multiError
	for _, c := range collectors {
		err := c.Collect(fileWriter)
		errors = appendError(errors, err)
	}

	if errors == nil {
		return nil
	}

	w, _ := fileWriter(errors)
	fmt.Fprint(w, errors)
	return errors
}

func Sonobuoy(cmd *cobra.Command, settings *env.Settings) error {
	fmt.Println("Generating sonobuoy configs and plugins")

	ns := settings.Namespace
	kc, _ := env.GetClient(settings)
	instance, _ := kc.GetInstance(Options.Instance, ns)
	c, _ := kube.GetKubeClient(settings.KubeConfig)

	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, ns)
	sbiConfig := sonobuoyInstanceConfig(instance, ov)
	sbcConfig := sonobuoyControllerConfig()
	sbpConfig := sonobuoyPluginConfig(c, instance, ov)
	err = saveConfigs(sbiConfig, sbcConfig, sbpConfig)
	return err
}

func sonobuoyInstanceConfig(instance *v1beta1.Instance, ov *v1beta1.OperatorVersion) *sonobuoy.Config {
	cfg := sonobuoy.DefaultConfig
	cfg.Description = "Sonobuoy diagnostics for KUDO instance " + instance.Name
	cfg.UUID = uuid.New()
	cfg.Filters.Namespaces = instance.Namespace
	cfg.Filters.LabelSelector = labelKudoOperator + "=" + instance.Labels[labelKudoOperator]
	cfg.Limits.PodLogs.Namespaces = instance.Namespace
	cfg.Limits.PodLogs.LabelSelector = labelKudoOperator + "=" + instance.Labels[labelKudoOperator]
	cfg.Plugins = sonobuoy.Plugins{{Name: "cmd-executor"}}
	return &cfg
}

func sonobuoyControllerConfig() *sonobuoy.Config {
	cfg := sonobuoy.DefaultConfig
	cfg.Description = "Sonobuoy diagnostics for KUDO controller"
	cfg.UUID = uuid.New()
	cfg.Filters.Namespaces = nsKudoSystem
	cfg.Filters.LabelSelector = "app=" + appKudoManager
	cfg.Limits.PodLogs.Namespaces = nsKudoSystem
	cfg.Limits.PodLogs.LabelSelector = "app=" + appKudoManager
	cfg.Plugins = sonobuoy.Plugins{}
	cfg.PluginSearchPath = []string{}
	return &cfg
}

func sonobuoyPluginConfig(c *kube.Client, instance *v1beta1.Instance, ov *v1beta1.OperatorVersion) *sonobuoy.PluginConfig {
	cfg := sonobuoy.DefaultPluginConfig
	cfg.Container.Args = pluginArgs(c, instance, ov)
	return &cfg
}

func pluginArgs(c *kube.Client, instance *v1beta1.Instance, ov *v1beta1.OperatorVersion) []string {
	ns, labels := instance.Namespace, labelKudoOperator + "=" + instance.Labels[labelKudoOperator]
	var args []string
	for _, r := range ov.Spec.Diagnostics.Bundle.Resources {
		switch r.Kind {
		case "Copy":
			args = append(args, convertCopyToArgs(r, ns, labels, )...)
		case "Command":
			args = append(args, convertCommandToArgs(r, ns, labels)...)
		case "Request":
			svcName := r.Spec.HttpSpec.Name
			svc, _ := c.KubeClient.CoreV1().Services(instance.Namespace).Get(svcName, metav1.GetOptions{}) // TODO: collect error
			args = append(args, convertRequestToArgs(svc, r)...)
		default:
			// TODO: collect error
		}
	}
	return args
}

func convertCopyToArgs(r v1beta1.DiagnosticResource, ns, labels string) []string {
	spec := r.Spec.CopySpec
	var args []string
	kind := strings.ToLower(r.Kind)
	for _, path := range spec.Paths {
		cmd := "cat " + path
		args = append(args, kind)
		args = append(args, cmd)
		args = append(args, ns)
		args = append(args, labels)
		args = append(args, fmt.Sprintf("%s_%s.txt", kind, r.Name))
	}
	return args
}

// TODO: refactor DRY
func convertCommandToArgs(r v1beta1.DiagnosticResource, ns, labels string) []string {
	spec := r.Spec.CommandSpec
	var args []string
	kind := strings.ToLower(r.Kind)
	for _, cmd := range spec.Commands {
		args = append(args, kind)
		args = append(args, cmd)
		args = append(args, ns)
		args = append(args, labels)
		args = append(args, fmt.Sprintf("%s_%s.txt", kind, r.Name))
	}
	return args
}

func convertRequestToArgs(svc *corev1.Service, r v1beta1.DiagnosticResource) []string {
	spec := r.Spec.HttpSpec
	i := 0
	// TODO: use named port
	for  ; i<len(svc.Spec.Ports) && int32(spec.Port)!=svc.Spec.Ports[i].Port; i++ {
	}
	if i == len(svc.Spec.Ports) {
		// TODO: handle error, requested port not found on this service
		return nil
	}
	svcPort := svc.Spec.Ports[i]
	if svcPort.Protocol != "TCP" {
		// TODO: throw error, unsupported protocol
		return nil
	}
	svcFqdn := fmt.Sprintf("%s.%s.%s", svc.Name, svc.Namespace, "svc.cluster.local")
	var cmd string
	switch spec.Method { //TODO: add to CRD!
	// TODO: how can I distinguish between http and https?
	case "GET":
		u := &url.URL{
			Scheme: "https",
			Host: fmt.Sprintf("%s:%d", svcFqdn, svcPort.Port),
			Path: spec.Path,
			RawQuery: spec.Query, // TODO: add to CRD
		}
		cmd = "curl -X GET " + u.String()
	case "HEAD":
		u := &url.URL{
			Scheme: "https",
			Host: fmt.Sprintf("%s:%d", svcFqdn, svcPort.Port),
			Path: spec.Path,
		}
		cmd = "curl -X GET " + u.String()
	case "":
		cmd = fmt.Sprintf("echo %s | nc %s %d", spec.Query, svcFqdn, svcPort.Port)
	default: // TODO: throw error unsupported method for diagnostics -
		return nil
	}
	kind := strings.ToLower(r.Kind)
	return []string {kind, cmd, fmt.Sprintf("%s_%s.txt", kind, r.Name)}
}

// TODO: pass and save accumulated errors
func saveConfigs(sbiConfig *sonobuoy.Config, sbcConfig *sonobuoy.Config, sbpConfig *sonobuoy.PluginConfig) error {
	dirName := "./sonobuoy"
	_ = os.Mkdir(dirName, 0700) // TODO: handle error
	iw, _ := os.Create(dirName + "/config.json")
	e := json.NewEncoder(iw)
	e.SetEscapeHTML(false)
	_ = e.Encode(sbiConfig)
	defer iw.Close()
	sw, _ := os.Create(dirName + "/config_kudo.json")
	_ = json.NewEncoder(sw).Encode(sbcConfig)
	defer sw.Close()
	pw, _ := os.Create(dirName + "/plugin.yaml")
	j,_ := json.Marshal(sbpConfig) // use JSON tags
	y, _ := yaml.JSONToYAML(j)
	pw.Write(y)
	defer pw.Close()
	return nil
}