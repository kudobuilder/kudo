package diagnostics

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"log"
)

const (
	ns_yml = `
apiVersion: v1
kind: Namespace
metadata:
  name: kudo-diag
`
	svcAccount_yml = `
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: kudo-diag
  name: kudo-diag-serviceaccount
  namespace: kudo-diag
`
	crb_yml = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    component: kudo-diag
    namespace: kudo-diag
  name: kudo-diag-serviceaccount-kudo-diag
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kudo-diag-serviceaccount-kudo-diag
subjects:
  - kind: ServiceAccount
    name: kudo-diag-serviceaccount
    namespace: kudo-diag
`
	cRole_yml = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    component: kudo-diag
    namespace: kudo-diag
  name: kudo-diag-serviceaccount-kudo-diag
rules:
  - apiGroups:
      - '*'
    resources:
      - '*'
    verbs:
      - '*'
  - nonResourceURLs:
      - '/metrics'
      - '/logs'
      - '/logs/*'
    verbs:
      - 'get'
`

	pod_yml = `
apiVersion: v1
kind: Pod
metadata:
  labels:
    component: kudo-diag
    namespace: kudo-diag
    tier: diagnostics 
  name: plugin-pod	
spec:
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  priority: 0
  restartPolicy: Never
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: kudo-diag-serviceaccount
  serviceAccountName: kudo-diag-serviceaccount
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
  - key: kubernetes.io/e2e-evict-taint-key
    operator: Exists
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - emptyDir: {}
    name: results
`

  //- name: kudo-diag-serviceaccount-token-82dqc
  //  secret:
  //    defaultMode: 420
  //    secretName: kudo-diag-serviceaccount-token-82dqc

//- args:
//- copy
//- cat /conf/zoo.cfg
//- default
//- kudo.dev/operator=zookeeper
//- copy_zookeeper-configuration.txt
//- command
//- nslookup google.com
//- default
//- kudo.dev/operator=zookeeper
//- command_dns-information.txt
//- request
//- echo stat | nc zookeeper-instance-cs.default.svc.cluster.local 2181
//- request_cs-stat.txt

	example_container_yml = `
command:
- ./run.sh
image: vvy/easy-sonobuoy-cmds:v0.1
imagePullPolicy: IfNotPresent
name: cmd-executor
resources: {}
terminationMessagePath: /dev/termination-log
terminationMessagePolicy: File
volumeMounts:
- mountPath: /tmp/results
  name: results
`

    //- mountPath: /var/run/secrets/kubernetes.io/serviceaccount
    //  name: sonobuoy-serviceaccount-token-82dqc
    //  readOnly: true

	example_container2_yml = `
  command:
  - sleep
  image: busybox
  imagePullPolicy: IfNotPresent
  name: sleeping-bb
  resources: {}
  terminationMessagePath: /dev/termination-log
  terminationMessagePolicy: File
`
)

func mustUnmarshal(yml string, v interface{}) interface{} {
	nsJson, err :=yaml.YAMLToJSON([]byte(yml))
	if err == nil {
		err = json.Unmarshal(nsJson, v)
	}
	fmt.Println(string(nsJson))
	if err!=nil {
		log.Fatal("invalid data for the plugin ", err)
	}
	return v
}

func createPluginPod(c *kube.Client, args []string) (*v1.Pod, error) {
	ns := mustUnmarshal(ns_yml, &v1.Namespace{}).(*v1.Namespace)
	svcAcc := mustUnmarshal(svcAccount_yml, &v1.ServiceAccount{}).(*v1.ServiceAccount)
	crb := mustUnmarshal(crb_yml, &rbacv1.ClusterRoleBinding{}).(*rbacv1.ClusterRoleBinding)
	cRole := mustUnmarshal(cRole_yml, &rbacv1.ClusterRole{}).(*rbacv1.ClusterRole)
	container := mustUnmarshal(example_container_yml, &v1.Container{}).(*v1.Container)
	//container.Args = append(container.Args, "1000")
	container.Args = args
	pod := mustUnmarshal(pod_yml, &v1.Pod{}).(*v1.Pod)
	pod.Spec.Containers = append(pod.Spec.Containers, *container)

	nsName := ns.Name
	var err error
	ns, err = c.KubeClient.CoreV1().Namespaces().Create(ns)
	if err != nil {
		log.Println(err)
	}
	svcAcc, err = c.KubeClient.CoreV1().ServiceAccounts(nsName).Create(svcAcc)
	if err != nil {
		log.Println(err)
	}
	cRole, err = c.KubeClient.RbacV1().ClusterRoles().Create(cRole)
	if err != nil {
		log.Println(err)
	}
	crb, err = c.KubeClient.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil {
		log.Println(err)
	}
	pod, err = c.KubeClient.CoreV1().Pods(nsName).Create(pod)
	if err != nil {
		log.Fatal(err)
	}
	//os.Exit(0)
	return pod, nil
}