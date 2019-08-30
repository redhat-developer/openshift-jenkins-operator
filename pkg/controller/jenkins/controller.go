package jenkins

import (
	"context"

	"github.com/go-logr/logr"

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
	// Create a new controller
	c, err := controller.New("jenkins-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Jenkins
	err = c.Watch(&source.Kind{Type: &jenkinsv1alpha1.Jenkins{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DeploymentConfig and requeue the owner Jenkins
	err = c.Watch(&source.Kind{Type: &appsv1.DeploymentConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.Jenkins{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource ImageStream and requeue the owner Jenkins
	err = c.Watch(&source.Kind{Type: &imagev1.ImageStream{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.Jenkins{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource ServiceAccount and requeue the owner Jenkins
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.Jenkins{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Route and requeue the owner Jenkins
	err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.Jenkins{},
	})
	if err != nil {
		return err
	}
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
		Name:     "web",
		Port:     80,
		Protocol: "TCP",
		TargetPort: intstr.IntOrString{
			IntVal: 8080,
			StrVal: "8080",
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
			Type:     "ClusterIP",
		},
	}

	return svc
}
