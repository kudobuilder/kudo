package bundle

// Framework is a representation of the KEP-9 Framework YAML
type Framework struct {
	Name              string          `json:"name"`
	Description       string          `json:"description,omitempty"`
	Version           string          `json:"version"`
	KUDOVersion       string          `json:"kudoVersion,omitempty"`
	KubernetesVersion string          `json:"kubernetesVersion,omitempty"`
	Maintainers       []string        `json:"maintainers,omitempty"`
	URL               string          `json:"url,omitempty"`
	Tasks             map[string]Task `json:"tasks"`
	Plans             map[string]Plan `json:"plans"`
}

// Task is a representation of the KEP-9 Task inside of Framework
type Task struct {
	Resources []string `json:"resources"`
}

// Plan is a representation of the KEP-9 Plan inside of Framework
type Plan struct {
	Strategy string  `json:"string,omitempty"`
	Phases   []Phase `json:"phases'"`
}

// Phase is a representation of the KEP-9 Phase inside of Framework
type Phase struct {
	Name     string `json:"name"`
	Strategy string `json:"string,omitempty"`
	Steps    []Step `json:"steps"`
}

// Step is a representation of the KEP-9 Step inside of Framework
type Step struct {
	Name   string   `json:"name"`
	Tasks  []string `json:"tasks"`
	Delete bool     `json:"delete,omitempty"`
}
