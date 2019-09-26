package jenkins

import (
	"fmt"
	"reflect"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	common "github.com/redhat-developer/openshift-jenkins-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	logController      = logf.Log.WithName("jenkins/controller.go")
	controllerMessages = common.NewMessages("Jenkins Controller")
)

const JenkinsControllerName = "jenkins-controller"

type ControlledResources struct {
	JenkinsInstance       *jenkinsv1alpha1.Jenkins
	DeploymentConfig      *appsv1.DeploymentConfig
	ImageStream           *imagev1.ImageStream
	JenkinsService        *corev1.Service
	JNLPService           *corev1.Service
	PersistentVolumeClaim *corev1.PersistentVolumeClaim
	Route                 *routev1.Route
	RoleBinding           *rbacv1.RoleBinding
	ServiceAccount        *corev1.ServiceAccount
}
type NamedResource struct {
	Object interface{}
	Name   string
}
type RuntimeResource struct {
	Object runtime.Object
	types.NamespacedName
}
type StaticResource struct {
	Object metav1.Object
	types.NamespacedName
}

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	JenkinsReconciler := newReconciler(mgr, *controllerMessages)
	return add(mgr, JenkinsReconciler)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new Jenkins Controller
	controllerMessages.LogInfo("Creating Jenkins Controller", logController)
	c, err := controller.New(JenkinsControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		controllerMessages.LogError(err, "Failed at creation of controller", logController)
		return err
	}

	// Create owner reference stating the owner of all the resources under the controller
	ownerRef := &jenkinsv1alpha1.Jenkins{}

	resourcesToWatch := []NamedResource{
		NamedResource{ownerRef, ""},
		NamedResource{&appsv1.DeploymentConfig{}, ""},
		NamedResource{&imagev1.ImageStream{}, ""},
		NamedResource{&corev1.ServiceAccount{}, ""},
		NamedResource{&corev1.PersistentVolumeClaim{}, ""},
		NamedResource{&routev1.Route{}, ""},
		NamedResource{&rbacv1.RoleBinding{}, ""},
		NamedResource{&corev1.ServiceAccount{}, ""},
	}

	for _, resource := range resourcesToWatch {
		ownerReference := ownerRef
		if reflect.DeepEqual(resource.Object, ownerRef) {
			ownerReference = nil
		}
		watchResourceOrStackError(c, resource, ownerReference)
	}
	return nil
}

// Setup Scheme for a resources
func watchResourceOrStackError(controller controller.Controller, resource NamedResource, owner runtime.Object) {
	namespaceNameLog := "| Namespace " + resource.Name + " | Name "
	message := "watchResourceOrStackError\n" + namespaceNameLog

	controllerMessages.LogInfo(message, logController)
	err := controller.Watch(&source.Kind{Type: resource.Object.(runtime.Object)}, &handler.EnqueueRequestForObject{})
	if err != nil {
		controllerMessages.LogError(err, "Cannot watch component", logController)
	} else {
		controllerMessages.LogInfo(fmt.Sprintf("Component %v of parent type %v is now being watched", resource.Object, owner), logController)
	}
}
