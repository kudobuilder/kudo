package tools

import (
	"flag"
	"github.com/kudobuilder/kudo/pkg/test"
	"testing"
)

func TestKudoFrameworks(t *testing.T) {
	path := "../../pkg/test/integration-tests/"

	args := flag.Args()

	if len(args) > 0 {
		path = args[0]
	}

	test.RunHarness(path, t)
}
