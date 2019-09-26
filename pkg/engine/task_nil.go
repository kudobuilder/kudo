package engine

// NullTask is a task that accepts nothing and has no public or private members
// +k8s:deepcopy-gen=true
type NilTask struct {
}

// NullTask Run has no side effects and returns no error. Useful for testing other workflows that require tasks.
func (n *NilTask) Run() error {
	return nil
}
