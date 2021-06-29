package controllers_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/controllers"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/controllers/controllersfakes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/controllers/remote"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Unit tests for BYOMachine Controller", func() {
	var (
		reconciler          *controllers.ByoMachineReconciler
		clientFake          *controllersfakes.FakeClient
		groupResource       schema.GroupResource
		byoMachineName      = "test-machine"
		byoMachineNamespace = "test-ns"
		err                 error
	)

	BeforeEach(func() {

		clientFake = &controllersfakes.FakeClient{}
		groupResource = schema.GroupResource{
			Group:    "infrastructure.cluster.x-k8s.io",
			Resource: "Byomachine",
		}

		reconciler = &controllers.ByoMachineReconciler{
			Client:  clientFake,
			Log:     ctrl.Log.WithName("controllers").WithName("ByoMachine"),
			Tracker: remote.NewTestClusterCacheTracker(log.NullLogger{}, clientFake, scheme.Scheme, client.ObjectKey{Name: "test-cluster", Namespace: "test-ns"}),
		}

	})

	It("should not error when byomachine is not present", func() {
		clientFake.GetReturns(apierrors.NewNotFound(groupResource, byoMachineName))

		byoMachineLookupkey := types.NamespacedName{Name: "fake-machine", Namespace: byoMachineNamespace}
		request := reconcile.Request{NamespacedName: byoMachineLookupkey}
		_, err = reconciler.Reconcile(context.TODO(), request)
		Expect(err).To(Not(HaveOccurred()))
	})

})
