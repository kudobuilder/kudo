package task

// NilTask is a task that accepts nothing and has no public or private members
type NilTask struct{}

// NilTask Run has no side effects and returns no error. Useful for testing other workflows that require tasks.
func (nt NilTask) Run(ctx Context) (bool, error) {
	return true, nil
}
