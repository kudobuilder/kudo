package task

import (
	"testing"
)

func TestTask_Run(t1 *testing.T) {
	type fields struct {
		Name string
		Kind string
		Spec TaskSpec
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Run Apply Task",
			fields: fields{
				Name: "Install",
				Kind: "Apply",
				Spec: TaskSpec{
					ApplyTask: ApplyTask{
						Resources: []string{},
					},
				},
			},
		},
		{
			name: "Run Nil Task",
			fields: fields{
				Name: "Do Nothing",
				Kind: "Nil",
				Spec: TaskSpec{
					DummyTask: DummyTask{},
				},
			},
		},
	}
	ctx := Context{}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Task{
				Name: tt.fields.Name,
				Kind: tt.fields.Kind,
				Spec: tt.fields.Spec,
			}
			if err := t.Run(ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
