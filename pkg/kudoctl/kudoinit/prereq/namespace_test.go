package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
)

func TestPrereq_Fail_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	init := NewInitializer(kudoinit.NewOptions("", "customNS", "", make([]string, 0)))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewError("Namespace customNS does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand"), result)
}

func TestPrereq_Ok_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	mockGetNamespace(client, "customNS")

	init := NewInitializer(kudoinit.NewOptions("", "customNS", "", make([]string, 0)))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewResult(), result)
}
