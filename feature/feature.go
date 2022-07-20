// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package feature

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

var (
	MutableGates featuregate.MutableFeatureGate = featuregate.NewFeatureGate()
	Gates        featuregate.FeatureGate        = MutableGates
)

func init() {
	runtime.Must(MutableGates.Add(defaultClusterAPIBYOHFeatureGates))
}

// defaultClusterAPIBYOHFeatureGates consists of all known cluster-api-byoh feature keys.
// To add a new feature, define a key for it above and add it here.
var defaultClusterAPIBYOHFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	// TODO-OBSERVABILITY - Task2
	// Add feature flag for Observability
}
