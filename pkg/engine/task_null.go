package engine

// NullTask is a task that accepts nothing and has no public or private members
// +k8s:deepcopy-gen=true
type NullTask struct {
}

// NullTask Run has no side effects and returns no error. Useful for testing other workflows that require tasks.
func (n *NullTask) Run() error {
	return nil
}
