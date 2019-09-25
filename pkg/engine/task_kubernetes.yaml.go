package engine

type KubernetesTask struct {
	Op     string
	Params map[string]interface{}
}

func Run() error {
	return nil
	// setup Kubernetes client
}
