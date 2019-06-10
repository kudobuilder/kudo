package vars

// Variables for flags
var (
	AllDependencies bool
	AutoApprove     bool
	Instance        string
	KubeConfigPath  string
	Namespace       string
	Parameter       []string
	PackageVersion  string
	StorageBucket   string
	StoragePrefix   string

		//FrameworkImportPath defines the location of the helm or KUDO framework definition that should be imported
	FrameworkImportPath string

	//Format specifies json or yaml to be exported
	Format string
)
