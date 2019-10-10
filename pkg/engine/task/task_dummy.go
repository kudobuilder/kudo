package task

import "errors"

// DummyTask is a task that can fail or succeed on demand
type DummyTask struct {
	Fail bool
}

// DummyTask Run has no side effects and returns a dummy error if Fail is true
func (dt DummyTask) Run(ctx Context) (bool, error) {
	if dt.Fail {
		return false, errors.New("dummy error")
	}
	return true, nil
}
