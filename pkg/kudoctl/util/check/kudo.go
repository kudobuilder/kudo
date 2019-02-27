package check

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/k8s"
	"github.com/pkg/errors"
)

func KudoCRDs(k *k8s.K2oClient) error {
	err := k.CRDsInstalled()
	if err != nil {
		return errors.WithMessage(err, "missing crd")
	}
	return nil
}
