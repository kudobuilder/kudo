package task

import (
	"reflect"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

func TestBuild(t *testing.T) {
	type args struct {
		task *v1alpha1.Task
	}
	tests := []struct {
		name    string
		args    args
		want    Tasker
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Build(tt.args.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
		})
	}
}
