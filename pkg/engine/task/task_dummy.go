package task

import (
	"errors"
	"fmt"
)

// DummyTask is a task that can fail or succeed on demand
type DummyTask struct {
	Name    string
	WantErr bool
	Fatal   bool
	Done    bool
}

// Run method for the tDummyTask. It has no side effects and returns a dummy error if WantErr is true
func (dt DummyTask) Run(ctx Context) (bool, error) {
	if dt.WantErr {
		err := errors.New("dummy error")
		if dt.Fatal {
			err = fmt.Errorf("fatal %w", ErrFatalExecution)
		}
		return false, err
	}

	return dt.Done, nil
}
