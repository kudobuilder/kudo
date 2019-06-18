// +build build_harness

package tools

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/test"
)

func init() {
	test.RegisterFlags()
}

func TestKudoFrameworks(t *testing.T) {
	test.HarnessFromFlags(t).Run()
}
