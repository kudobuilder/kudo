package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verify"
)

func TestPrereq_Fail_PreValidate_CustomServiceAccount(t *testing.T) {
	client := getFakeClient()

	init := NewInitializer(kudoinit.NewOptions("", "", "customSA", make([]string, 0)))

	result := init.PreInstallVerify(client)

	assert.EqualValues(t, verify.NewError("Service Account customSA does not exists - KUDO expects the serviceAccount to be present in the namespace kudo-system"), result)
}

func TestPrereq_Fail_PreValidate_CustomServiceAccount_MissingPermissions(t *testing.T) {
	client := getFakeClient()

	customSA := "customSA"

	mockListServiceAccounts(client, customSA)

	init := NewInitializer(kudoinit.NewOptions("", "", customSA, make([]string, 0)))

	result := init.PreInstallVerify(client)

	assert.EqualValues(t, verify.NewError("Service Account customSA does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace kudo-system and to have cluster-admin role"), result)
}

func TestPrereq_Ok_PreValidate_CustomServiceAccount(t *testing.T) {
	client := getFakeClient()

	customSA := "customSA"
	opts := kudoinit.NewOptions("", "", customSA, make([]string, 0))

	mockListServiceAccounts(client, opts.ServiceAccount)
	mockListClusterRoleBindings(client, opts)

	init := NewInitializer(opts)
	result := init.PreInstallVerify(client)

	assert.EqualValues(t, verify.NewResult(), result)
}
