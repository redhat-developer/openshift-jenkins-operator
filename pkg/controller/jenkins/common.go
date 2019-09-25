package jenkins

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Done when no error is informed and request is not set for requeue.
func Done() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}
