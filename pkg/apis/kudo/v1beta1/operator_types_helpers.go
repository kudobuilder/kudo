package v1beta1

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOperator(name, ns string, c client.Client) (*Operator, error) {
	o := &Operator{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
