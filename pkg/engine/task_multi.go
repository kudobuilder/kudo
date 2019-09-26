package engine

import "fmt"

type MultiTask struct {
	InitialInput interface{}
	Tasks        []TaskBuilder
	lastOutput   interface{}
}

func (m *MultiTask) Run(ctx Context) error {
	var input interface{}
	input = m.InitialInput

	for i, t := range m.Tasks {
		task, err := t(input)
		if err != nil {
			return err
		}

		err = task.Run(ctx)
		if err != nil {
			return err
		}

		op, ok := task.(Outputter)
		if ok {
			input = op.Output()
		}

		fmt.Println(i, len(m.Tasks))
		if i+1 == len(m.Tasks) && ok {
			m.lastOutput = input
		}
	}

	return nil
}

func (m *MultiTask) Output() interface{} {
	return m.lastOutput
}
