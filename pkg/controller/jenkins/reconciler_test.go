package jenkins

import (
	"context"
	"fmt"
	"testing"

	"github.com/redhat-developer/openshift-jenkins-operator/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	reconcilerNs   = "testing"
	reconcilerName = "jenkins"
)

func init() {
	logf.SetLogger(logf.ZapLogger(true))
}

// reconcileRequest creates a reconcile.Request object using global variables.
func reconcileRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: reconcilerNs,
			Name:      reconcilerName,
		},
	}
}

// TestReconcilerForJenkinsPodCreation
// and using TriggerRebind = true, false
func TestReconcilerForJenkinsPodCreation(t *testing.T) {
	ctx := context.TODO()
	f := mocks.NewFake(t, reconcilerNs)

	fakeClient := f.FakeClient()
	reconciler := &JenkinsReconciler{client: fakeClient, dynClient: f.FakeDynClient(), scheme: f.S}

	t.Run("reconcile", func(t *testing.T) {
		res, err := reconciler.Reconcile(reconcileRequest())
		assert.Nil(t, err)
		assert.False(t, res.Requeue)

		namespacedName := types.NamespacedName{Namespace: reconcilerNs, Name: reconcilerName}
		d := appsv1.Deployment{}
		require.Nil(t, fakeClient.Get(ctx, namespacedName, &d))

		containers := d.Spec.Template.Spec.Containers
		require.Equal(t, 1, len(containers))
		require.Equal(t, 1, len(containers[0].EnvFrom))
		assert.NotNil(t, containers[0].EnvFrom[0].SecretRef)
		assert.Equal(t, reconcilerName, containers[0].EnvFrom[0].SecretRef.Name)

		sbrOutput := v1alpha1.ServiceBindingRequest{}
		require.Nil(t, fakeClient.Get(ctx, namespacedName, &sbrOutput))
		require.Equal(t, "Success", sbrOutput.Status.BindingStatus)
		require.Equal(t, reconcilerName, sbrOutput.Status.Secret)

		require.Equal(t, 1, len(sbrOutput.Status.ApplicationObjects))
		expectedStatus := fmt.Sprintf("%s/%s", reconcilerNs, reconcilerName)
		assert.Equal(t, expectedStatus, sbrOutput.Status.ApplicationObjects[0])
	})

}
