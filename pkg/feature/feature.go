package feature

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

var (
	DefaultMutableFeatureGate featuregate.MutableFeatureGate = featuregate.NewFeatureGate()

	DefaultFeatureGate featuregate.FeatureGate = DefaultMutableFeatureGate
)

func init() {
	runtime.Must(DefaultMutableFeatureGate.Add(defaultKUDOFeatureGates))
}

var defaultKUDOFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{}
