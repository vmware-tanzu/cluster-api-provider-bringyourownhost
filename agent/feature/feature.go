package feature
import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // owner: @username
	// // alpha: v1.X
	// MyFeature featuregate.Feature = "MyFeature".

	// MachinePool is a feature gate for MachinePool functionality.
	//
	// alpha: v0.3
	// MachinePool featuregate.Feature = "MachinePool"

	// ClusterResourceSet is a feature gate for the ClusterResourceSet functionality.
	//
	// alpha: v0.3
	// beta: v0.4
	// ClusterResourceSet featuregate.Feature = "ClusterResourceSet"

	// ClusterTopology is a feature gate for the ClusterClass and managed topologies functionality.
	//
	// alpha: v0.4
	// ClusterTopology featuregate.Feature = "ClusterTopology"

	SecureAccess featuregate.Feature = "SecureAccess"
)

var (
	// MutableGates is a mutable version of DefaultFeatureGate.
	// Only top-level commands/options setup and the k8s.io/component-base/featuregate/testing package should make use of this.
	// Tests that need to modify featuregate gates for the duration of their test should use:
	//   defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.<FeatureName>, <value>)()
	MutableGates featuregate.MutableFeatureGate = featuregate.NewFeatureGate()

	// Gates is a shared global FeatureGate.
	// Top-level commands/options setup that needs to modify this featuregate gate should use DefaultMutableFeatureGate.
	Gates featuregate.FeatureGate = MutableGates
)

func init() {
	runtime.Must(MutableGates.Add(defaultClusterAPIFeatureGates))
}

// defaultClusterAPIFeatureGates consists of all known cluster-api-specific feature keys.
// To add a new feature, define a key for it above and add it here.
var defaultClusterAPIFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	// Every feature should be initiated here:
	// MachinePool:        {Default: false, PreRelease: featuregate.Alpha},
	// ClusterResourceSet: {Default: true, PreRelease: featuregate.Beta},
	// ClusterTopology:    {Default: false, PreRelease: featuregate.Alpha},
	SecureAccess: {Default: false, PreRelease: featuregate.Alpha},
}