package engine

type MultiTask struct {
	InitialInput interface{}
	Tasks        []TaskBuilder
	lastOutput   interface{}
}

func (m *MultiTask) Run(ctx Context) error {
	var input interface{}
	input = m.InitialInput

	for _, t := range m.Tasks {
		task, err := t(input)
		if err != nil {
			return err
		}

		err = task.Run(ctx)
		if err != nil {
			return err
		}

		if op, ok := task.(Outputter); ok {
			input = op.Output()
		}
	}

	return nil
}

func (m *MultiTask) Output() interface{} {
	return m.lastOutput
}
