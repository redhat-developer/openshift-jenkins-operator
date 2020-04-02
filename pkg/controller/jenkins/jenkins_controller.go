package jenkins

import (
	"reflect"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	common "github.com/redhat-developer/openshift-jenkins-operator/pkg/common"
	kappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	j "github.com/redhat-developer/openshift-jenkins-operator/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logController      = logf.Log.WithName("jenkins/jenkins_controller.go")
	controllerMessages = common.NewMessages("Jenkins Controller")
)

const JenkinsControllerName = "jenkins-controller"

type ControlledResources struct {
	JenkinsInstance       *jenkinsv1alpha1.Jenkins
	DeploymentConfig      *appsv1.DeploymentConfig
	Deployment            *kappsv1.Deployment
	ImageStream           *imagev1.ImageStream
	JenkinsService        *corev1.Service
	JNLPService           *corev1.Service
	Volume                *corev1.Volume
	PersistentVolumeClaim *corev1.PersistentVolumeClaim `json:",omitempty" bson:",omitempty"`
	Route                 *routev1.Route
	RoleBinding           *rbacv1.RoleBinding
	ServiceAccount        *corev1.ServiceAccount
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

	resourcesToWatch := []j.NamedResource{
		j.NamedResource{ownerRef, ""},
		j.NamedResource{&appsv1.DeploymentConfig{}, ""},
		j.NamedResource{&kappsv1.Deployment{}, ""},
		j.NamedResource{&imagev1.ImageStream{}, ""},
		j.NamedResource{&corev1.ServiceAccount{}, ""},
		j.NamedResource{&corev1.PersistentVolumeClaim{}, ""},
		j.NamedResource{&routev1.Route{}, ""},
		j.NamedResource{&rbacv1.RoleBinding{}, ""},
		j.NamedResource{&corev1.ServiceAccount{}, ""},
	}

	for _, resource := range resourcesToWatch {
		ownerReference := ownerRef
		if reflect.DeepEqual(resource.Object, ownerRef) {
			ownerReference = nil
		}
		j.WatchResourceOrStackError(c, resource, ownerReference)
	}
	return nil
}

