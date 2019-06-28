package template

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes/scheme"
)

//ParseKubernetesObjects parses a list of runtime.Objects from the provided yaml
func ParseKubernetesObjects(yaml string) (objs []runtime.Object, err error) {
	sepYamlfiles := strings.Split(yaml, "---")
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, e := decode([]byte(f), nil, nil)

		if e != nil {
			err = e
			return
		}
		objs = append(objs, obj)
	}
	return
}
