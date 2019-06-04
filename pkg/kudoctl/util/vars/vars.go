package vars

// Variables for flags
var (
	AllDependencies      bool
	AutoApprove          bool
	GithubCredentialPath string
	GithubCredentials    string
	Instance             string
	KubeConfigPath       string
	Namespace            string
	Parameter            []string
	PackageVersion       string
	StorageBucket        string
	StoragePrefix        string
	RepoPath 			 string = "$HOME/.kudo/repository" // this won't work on windows
)
