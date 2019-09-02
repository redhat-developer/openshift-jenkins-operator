package jenkins

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	gerrors "github.com/pkg/errors"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("jenkins/controller.go")

const (
	// NamespaceDefault means the object is in the default namespace which is applied when not specified by clients
	JenkinsControllerName  = "jenkins-controller"
	JenkinsWebPortName     = "web"
	JenkinsWebPortProtocol = corev1.ProtocolTCP
	JenkinsWebPort         = 80
	JenkinsWebPortAsInt    = 8080
	JenkinsWebPortAsStr    = "8080"
)

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJenkins{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

// blank assignment to verify that ReconcileJenkins implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkins{}

// ReconcileJenkins reconciles a Jenkins object
type ReconcileJenkins struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	logger logr.Logger
	result reconcile.Result
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

// Reconcile reads that state of the cluster for a Jenkins object and makes changes based on the state read
// and what is in the Jenkins.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkins) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.result = reconcile.Result{}
	r.logger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.logger.Info("Reconciling Jenkins")

	// Fetch the Jenkins instance
	instance := &jenkinsv1alpha1.Jenkins{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return r.result, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	dc := newDeploymentConfigForCR(instance)

	// Define Jenkins Services
	jenkinsPort := corev1.ServicePort{
		Name:     JenkinsWebPortName,
		Port:     JenkinsWebPort,
		Protocol: JenkinsWebPortProtocol,
		TargetPort: intstr.IntOrString{
			IntVal: JenkinsWebPortAsInt,
			StrVal: JenkinsWebPortAsStr,
		},
	}
	jenkinsJNLPPort := corev1.ServicePort{
		Name:     "agent",
		Port:     50000,
		Protocol: "TCP",
		TargetPort: intstr.IntOrString{
			IntVal: 50000,
			StrVal: "50000",
		},
	}
	jenkinsSvc := newJenkinsServiceForCR(instance, "jenkins", jenkinsPort)              // jenkins service
	jenkinsJNLPSvc := newJenkinsServiceForCR(instance, "jenkins-jnlp", jenkinsJNLPPort) // jenknis jnlp service

	// Set Jenkins instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DC already exists
	found := &appsv1.DeploymentConfig{}
	namespacedName := types.NamespacedName{
		Namespace: found.Namespace,
		Name:      found.Name,
	}

	//TODO implements error checking
	err = r.createResourceIfNotPresent(namespacedName, dc)
	err = r.createResourceIfNotPresent(namespacedName, jenkinsSvc)
	err = r.createResourceIfNotPresent(namespacedName, jenkinsJNLPSvc)
	if err != nil {
		return r.result, err
	}
	return r.result, nil
}

func (r *ReconcileJenkins) createResourceIfNotPresent(key types.NamespacedName, resource runtime.Object) error {
	err := r.client.Get(context.TODO(), key, resource)
	if err != nil && errors.IsNotFound(err) {
		r.logger.Info("Creating a new DeploymentConfig", "DeploymentConfig.Namespace", key.Namespace, "DeploymentConfig.Name", key.Name)
		err = r.client.Create(context.TODO(), resource)
		if err != nil {
			return err
		}
		// Resource created successfully - don't requeue
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// newDeploymentConfigForCR returns a jenkins DeploymentConfig with the same name/namespace as the cr
func newDeploymentConfigForCR(cr *jenkinsv1alpha1.Jenkins) *appsv1.DeploymentConfig {
	labels := map[string]string{
		"app":  cr.Name,
		"test": "redhat-developer",
	}
	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labels,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image-registry.openshift-image-registry.svc:5000/openshift/jenkins",
							Name:  "jenkins",
						},
					},
				},
			},
		},
	}

	return dc
}

func newJenkinsServiceForCR(cr *jenkinsv1alpha1.Jenkins, name string, port corev1.ServicePort) *corev1.Service {
	labels := map[string]string{
		"app":  cr.Name,
		"test": "redhat-developer",
	}
	ports := []corev1.ServicePort{port}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	return svc
}
