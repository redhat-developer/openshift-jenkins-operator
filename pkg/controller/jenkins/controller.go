package jenkins

import (
	"context"

	jenkinsv1alpha1 "github.com/akram/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	"github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_jenkins")

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
	err = c.Watch(&source.Kind{Type: &v1.DeploymentConfig{}}, &handler.EnqueueRequestForOwner{
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
}

// Reconcile reads that state of the cluster for a Jenkins object and makes changes based on the state read
// and what is in the Jenkins.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkins) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Jenkins")

	// Fetch the Jenkins instance
	instance := &jenkinsv1alpha1.Jenkins{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	dc := newDeploymentConfigForCR(instance)

	// Set Jenkins instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this DC already exists
	found := &v1.DeploymentConfig{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dc.Name, Namespace: dc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DeploymentConfig", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
		err = r.client.Create(context.TODO(), dc)
		if err != nil {
			return reconcile.Result{}, err
		}

		// DC created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// DC already exists - don't requeue
	reqLogger.Info("Skip reconcile: DeploymentConfig already exists", "DeploymentConfig.Namespace", found.Namespace, "DeploymentConfig.Name", found.Name)
	return reconcile.Result{}, nil
}

// newDeploymentConfigForCR returns a jenkins DeploymentConfig with the same name/namespace as the cr
func newDeploymentConfigForCR(cr *jenkinsv1alpha1.Jenkins) *v1.DeploymentConfig {
	labels := map[string]string{
		"app":  cr.Name,
		"test": "akram",
	}
	dc := &v1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: v1.DeploymentConfigSpec{
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
