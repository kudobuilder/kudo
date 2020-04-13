package sonobuoy

import (
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var resources = []string{
	"apiservices",
	"certificatesigningrequests",
	"clusterrolebindings",
	"clusterroles",
	"componentstatuses",
	"configmaps",
	"controllerrevisions",
	"cronjobs",
	"customresourcedefinitions",
	"daemonsets",
	"deployments",
	"endpoints",
	"ingresses",
	"jobs",
	"leases",
	"limitranges",
	"mutatingwebhookconfigurations",
	"namespaces",
	"networkpolicies",
	"nodes",
	"persistentvolumeclaims",
	"persistentvolumes",
	"poddisruptionbudgets",
	"pods",
	"podlogs",
	"podsecuritypolicies",
	"podtemplates",
	"priorityclasses",
	"replicasets",
	"replicationcontrollers",
	"resourcequotas",
	"rolebindings",
	"roles",
	"servergroups",
	"serverversion",
	"serviceaccounts",
	"services",
	"statefulsets",
	"storageclasses",
	"validatingwebhookconfigurations",
	"volumeattachments",

	"instances",
	"operators",
	"operatorversions",
}

/*
"Filters":{
"Namespaces":".*",
"LabelSelector":""
},
"Limits":{
"PodLogs":{
"Namespaces":"",
"SonobuoyNamespace":true,
"FieldSelectors":[

],
"LabelSelector":"",
"Previous":false,
"SinceSeconds":null,
"SinceTime":null,
"Timestamps":false,
"TailLines":null,
"LimitBytes":null,
"LimitSize":"",
"LimitTime":""
}
},
"QPS":30,
"Burst":50,
"Server":{
"bindaddress":"0.0.0.0",
"bindport":8080,
"advertiseaddress":"",
"timeoutseconds":10800
},
"Plugins":[

{
"name":"systemd-logs"
}
],
"PluginSearchPath":[
".\/plugins.d",
"\/etc\/sonobuoy\/plugins.d",
"~\/sonobuoy\/plugins.d"
],
"Namespace":"sonobuoy",
"WorkerImage":"sonobuoy\/sonobuoy:v0.17.2",
"ImagePullPolicy":"IfNotPresent",
"ImagePullSecrets":"",
"ProgressUpdatesPort":"8099"
*/

var DefaultConfig = Config{
	Description: "DEFAULT",
	//UUID:"54297938-57c3-4e94-b194-21e5fb8358b2",
	Version:    "v0.17.2",
	ResultsDir: "/tmp/sonobuoy",
	Resources:  resources,
	Filters: Filters{
		Namespaces:    ".*",
		LabelSelector: "",
	},
	Limits: Limits{
		PodLogs: PodLogs{
			Namespaces:        "",
			SonobuoyNamespace: false,
			FieldSelectors:    []string{},
			LabelSelector:     "",
			Previous:          false,
			SinceSeconds:      nil,
			SinceTime:         nil,
			Timestamps:        false,
			TailLines:         nil,
			LimitBytes:        nil,
			LimitSize:         "",
			LimitTime:         "",
		},
	},
	QPS:   30,
	Burst: 50,
	Server: Server{
		BindAddress:       "0.0.0.0",
		BindPort:          8080,
		AdvertisedAddress: "",
		TimeoutSeconds:    10800,
	},
	PluginSearchPath: []string{
		"./plugins.d",
		"/etc/sonobuoy/plugins.d",
		"~/sonobuoy/plugins.d",
	},
	Namespace:        "sonobuoy",
	WorkerImage:      "sonobuoy/sonobuoy:v0.17.2",
	ImagePullPolicy:  "IfNotPresent",
	ImagePullSecrets: "",
}

var DefaultPluginConfig = PluginConfig{
	Spec: v1.PodSpec{
		Containers:         []v1.Container{},
		RestartPolicy:      "Never",
		ServiceAccountName: "sonobuoy-serviceaccount",
		DNSPolicy:          "ClusterFirst",
		Tolerations: []v1.Toleration{
			{
				Effect:   v1.TaintEffectNoSchedule,
				Key:      "node-role.kubernetes.io/master",
				Operator: v1.TolerationOpExists,
			},
			{
				Key:      "CriticalAddonsOnly",
				Operator: v1.TolerationOpExists,
			},
			{
				Key:      "kubernetes.io/e2e-evict-taint-key",
				Operator: v1.TolerationOpExists,
			},
		},
	},
	Container: v1.Container{
		Command:   []string{"./run.sh"},
		Image:     "vvy/easy-sonobuoy-cmds:v0.1",
		Name:      "cmd-executor",
		Resources: v1.ResourceRequirements{},
		VolumeMounts: []v1.VolumeMount{
			{
				MountPath: "/tmp/results",
				Name:      "results",
			},
		},
	},
	SonobuoyConfig: DriverConfig{
		Driver:       "Job",
		PluginName:   "cmd-executor",
		ResultFormat: "raw",
	},
}

/*
spec:
args:
- req
- echo stat | nc zookeeper-instance-cs.default.svc.cluster.local 2181
- zoo_stat.txt
- cmd
- cat /conf/zoo.cfg
- default
- kudo.dev/instance=zookeeper-instance
- zoo.cfg
command:
- ./run.sh
image: vvy/easy-sonobuoy-cmds:v0.1
name: plugin
resources: {}
volumeMounts:
- mountPath: /tmp/results
name: results
*/

type Config struct {
	Description        string    //"DEFAULT",
	UUID               uuid.UUID // "54297938-57c3-4e94-b194-21e5fb8358b2",
	Version            string    //"v0.17.2",
	ResultsDir         string    /* TODO: what is it?*/ //:"\/tmp\/sonobuoy",
	Resources          []string
	Filters            Filters
	Limits             Limits
	QPS                int // TODO: what is it? //30,
	Burst              int //TODO: what is it? //50,
	Server             Server
	Plugins            Plugins
	PluginSearchPath   []string
	Namespace          string
	WorkerImage        string
	ImagePullPolicy    string // IfNotPresent //TODO: any K type for that?
	ImagePullSecrets   string
	ProgressUpdatePort string
}

type Filters struct {
	Namespaces    string // ".*"
	LabelSelector string
}

type Limits struct {
	PodLogs PodLogs
}

type PodLogs struct {
	Namespaces        string
	SonobuoyNamespace bool     //true
	FieldSelectors    []string // defaults to empty list, not null
	LabelSelector     string
	// Fields from PodLogOptions
	Previous     bool         //false
	SinceSeconds *int64       //null
	SinceTime    *metav1.Time //null
	Timestamps   bool         //false,
	TailLines    *int64       //null,
	LimitBytes   *int64       //null,
	LimitSize    string       //TODO: what is it?
	LimitTime    string       //TODO: what is it?
}

type Plugins []struct {
	Name string
}

type Server struct {
	BindAddress       string `json:"bindaddress"`      //"0.0.0.0"
	BindPort          int    `json:"bindport"`         // 8080
	AdvertisedAddress string `json:"advertiseaddress"` //""
	TimeoutSeconds    int    `json:"timeoutseconds"`   // 10800
}

type DriverConfig struct {
	Driver       string `json:"driver"`        // TODO: Job or ?
	PluginName   string `json:"plugin-name"`   // cmd-executor
	ResultFormat string `json:"result-format"` //TODO: raw or ?
}

type PluginConfig struct {
	Spec           v1.PodSpec   `json:"podSpec"`
	Container      v1.Container `json:"spec"`
	SonobuoyConfig DriverConfig `json:"sonobuoy-config"`
}
