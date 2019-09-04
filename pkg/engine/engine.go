package engine

// Engine is an interface for any renderable templating engine
type Engine interface {
	Render(string, map[string]interface{}) (string, error)
}
