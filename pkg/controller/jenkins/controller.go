package jenkins

import (
	"fmt"

	gerrors "github.com/pkg/errors"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("jenkins/controller.go")

const (
	// NamespaceDefault means the object is in the default namespace which is applied when not specified by clients
	JenkinsControllerName = "jenkins-controller"
)

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	//	errors := gerrors.New("Initialisation errors")

	// Create a new controller
	c, err := controller.New(JenkinsControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ownerRef := &jenkinsv1alpha1.Jenkins{}
	handler := &handler.EnqueueRequestForObject{}
	// Watch for changes to primary resource Jenkins
	err = c.Watch(&source.Kind{Type: ownerRef}, handler)
	if err != nil {
		return err
	}
	watchResourceOrStackError(c, ownerRef, nil)
	watchResourceOrStackError(c, &appsv1.DeploymentConfig{}, ownerRef) // Watch DeploymentConfig and requeue the owner Jenkins
	watchResourceOrStackError(c, &imagev1.ImageStream{}, ownerRef)
	watchResourceOrStackError(c, &corev1.ServiceAccount{}, ownerRef)
	watchResourceOrStackError(c, &routev1.Route{}, ownerRef)

	// TODO check if errors is empty or not
	return nil
}

// Setup Scheme for a resources
func watchResourceOrStackError(controller controller.Controller, resourceType runtime.Object, ownerType runtime.Object) error {
	// Watch for changes to  resource  and requeue the owner to owner
	err := controller.Watch(&source.Kind{Type: resourceType}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    ownerType,
	})
	if err != nil {
		gerrors.Wrap(err, "Cannot watch component")
	} else {
		log.Info(fmt.Sprintf("Component %v of parent type %v is now being watched", resourceType, ownerType))
	}
	return err
}
