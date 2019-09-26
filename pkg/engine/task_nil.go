package engine

// NilTask is a task that accepts nothing and has no public or private members
// +k8s:deepcopy-gen=true
type NilTask struct {
}

// NilTask Run has no side effects and returns no error. Useful for testing other workflows that require tasks.
func (n *NilTask) Run(ctx Context) error {
	return nil
}
