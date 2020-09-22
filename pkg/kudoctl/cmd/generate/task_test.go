package generate

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
)

func TestAddTask(t *testing.T) {
	goldenFile := "task"
	path := "/opt/zk"
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../../packages/testdata/zk", "/opt")

	task := fooTask()

	err := AddTask(fs, path, &task)
	assert.NoError(t, err)

	operator, err := afero.ReadFile(fs, "/opt/zk/operator.yaml")
	assert.NoError(t, err)

	gp := filepath.Join("testdata", goldenFile+".golden")

	if *updateGolden {
		t.Logf("updating golden file %s", goldenFile)

		//nolint:gosec
		if err := ioutil.WriteFile(gp, operator, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	golden, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, string(golden), string(operator), "for golden file: %s", gp)
}

func TestEnsureTaskResources(t *testing.T) {
	path := "/opt/zk"
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../../packages/testdata/zk", "/opt")

	task := fooTask()

	// ensure an apply task saves
	err := EnsureTaskResources(fs, path, &task)
	assert.NoError(t, err)

	ok, err := afero.Exists(fs, "/opt/zk/templates/bar.yaml")
	assert.NoError(t, err)
	assert.True(t, ok)

	// ensure pipe task saves
	task = pipeTask()
	err = EnsureTaskResources(fs, path, &task)
	assert.NoError(t, err)

	ok, err = afero.Exists(fs, "/opt/zk/templates/pipe-pod.yaml")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestAddTask_bad_path(t *testing.T) {
	path := "."
	fs := afero.OsFs{}

	task := fooTask()

	err := AddTask(fs, path, &task)
	assert.Error(t, err)
}

func fooTask() kudoapi.Task {
	res := kudoapi.ResourceTaskSpec{Resources: []string{"bar.yaml"}}
	task := kudoapi.Task{
		Name: "Foo",
		Kind: "Apply",
		Spec: kudoapi.TaskSpec{ResourceTaskSpec: res}}
	return task
}

func pipeTask() kudoapi.Task {
	res := kudoapi.PipeTaskSpec{Pod: "pipe-pod.yaml"}
	task := kudoapi.Task{
		Name: "Foo",
		Kind: "Pipe",
		Spec: kudoapi.TaskSpec{PipeTaskSpec: res}}
	return task
}

func TestListTaskNames(t *testing.T) {
	fs := afero.OsFs{}
	tasks, err := TaskList(fs, "../../packages/testdata/zk")
	assert.NoError(t, err)

	assert.Equal(t, 3, len(tasks))
	assert.Equal(t, "infra", tasks[0].Name)
}

func TestTaskInList(t *testing.T) {
	fs := afero.OsFs{}
	name := "app"
	check, err := TaskInList(fs, "../../packages/testdata/zk", name)
	assert.NoError(t, err)
	assert.True(t, check)

	name = "wth"
	check, err = TaskInList(fs, "../../packages/testdata/zk", name)
	assert.NoError(t, err)
	assert.False(t, check)
}
