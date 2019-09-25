package engine

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestTask_Run(t1 *testing.T) {
	type fields struct {
		Name       string
		Kind       string
		NullTask   NullTask
		CreateTask CreateTask
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Run Create Task",
			fields: fields{
				Name: "createSomeShit",
				Kind: "create",
				CreateTask: CreateTask{
					Resources: []runtime.Object{},
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Task{
				Name:       tt.fields.Name,
				Kind:       tt.fields.Kind,
				NullTask:   tt.fields.NullTask,
				CreateTask: tt.fields.CreateTask,
			}
			if err := t.Run(); (err != nil) != tt.wantErr {
				t1.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
