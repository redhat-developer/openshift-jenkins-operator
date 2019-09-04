package jenkins

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "github.com/openshift/api/apps/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// NamespaceDefault means the object is in the default namespace which is applied when not specified by clients
	JenkinsAppLabelName    = "app"
	JenkinsWebPortName     = "web"
	JenkinsWebPortProtocol = corev1.ProtocolTCP
	JenkinsWebPort         = 80
	JenkinsWebPortAsInt    = 8080
	JenkinsWebPortAsStr    = "8080"

	JenkinsAgentPortName     = "agent"
	JenkinsAgentPortProtocol = corev1.ProtocolTCP
	JenkinsAgentPort         = 50000
	JenkinsAgentPortAsInt    = 50000
	JenkinsAgentPortAsStr    = "50000"

	JenkinsServiceName       = "jenkins"
	JenkinsJNLPServiceName   = "jenkins-jnlp"
	JenkinsJnlpServiceSuffix = "-jnlp"
	JenkinsImage             = "image-registry.openshift-image-registry.svc:5000/openshift/jenkins"
	JenkinsContainerName     = "jenkins"
	JenkinsAppLabel          = "app"

	JenkinsPvcName         = "jenkins"
	JenkinsPvcSize         = "1Gi"
	JenkinsVolumeName      = "jenkins-data"
	JenkinsVolumeMountPath = "/var/lib/jenkins"
)

// ReconcileJenkins reconciles a Jenkins object
type JenkinsReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	dynClient dynamic.Interface
	scheme    *runtime.Scheme
	logger    logr.Logger
	result    reconcile.Result
}

// blank assignment to verify that ReconcileJenkins implements reconcile.Reconciler
var _ reconcile.Reconciler = &JenkinsReconciler{}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &JenkinsReconciler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// Reconcile reads that state of the cluster for a Jenkins object and makes changes based on the state read
// and what is in the Jenkins.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *JenkinsReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.result = reconcile.Result{}
	r.logger = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.logger.Info("Reconciling Jenkins")

	jenkinsInstanceName := request.Name

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

	// Define a new DC object
	dc := newDeploymentConfig(instance)

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
		Name:     JenkinsAgentPortName,
		Port:     JenkinsAgentPort,
		Protocol: JenkinsAgentPortProtocol,
		TargetPort: intstr.IntOrString{
			IntVal: JenkinsAgentPort,
			StrVal: JenkinsAgentPortAsStr,
		},
	}
	jenkinsSvc := newJenkinsService(instance, jenkinsInstanceName, jenkinsPort)                                  // jenkins service
	jenkinsJNLPSvc := newJenkinsService(instance, jenkinsInstanceName+JenkinsJnlpServiceSuffix, jenkinsJNLPPort) // jenknis jnlp service
	jenkinsPvc := newJenkinsPvc(instance, jenkinsInstanceName)                                                   // jenknis jnlp service

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
	err = r.createResourceIfNotPresent(namespacedName, jenkinsPvc)

	if err != nil {
		return r.result, err
	}
	return r.result, nil
}

func (r *JenkinsReconciler) createResourceIfNotPresent(key types.NamespacedName, resource runtime.Object) error {
	err := r.client.Get(context.TODO(), key, resource)
	if err != nil && errors.IsNotFound(err) {
		r.logger.Info("Creating a new Object", "in Namespace", key.Namespace, "Resource.Name", resource)
		err = r.client.Create(context.TODO(), resource)
		if err != nil {
			r.logger.Info("Error while creating an object", "Object.Namespace", key.Namespace, "Object.Name", resource, "Error:", err)
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
func newDeploymentConfig(cr *jenkinsv1alpha1.Jenkins) *appsv1.DeploymentConfig {
	labels := map[string]string{
		JenkinsAppLabelName: cr.Name,
	}
	jenkinsInstanceName := cr.Name
	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jenkinsInstanceName,
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
							Image: JenkinsImage,
							Name:  JenkinsContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{Name: JenkinsVolumeName, MountPath: JenkinsVolumeMountPath},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: JenkinsVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: jenkinsInstanceName},
							},
						},
					},
				},
			},
		},
	}

	return dc
}

// newJenkinsServicefor templates a new Service for Jenkins
func newJenkinsService(cr *jenkinsv1alpha1.Jenkins, name string, port corev1.ServicePort) *corev1.Service {
	labels := map[string]string{JenkinsAppLabel: cr.Name}
	ports := []corev1.ServicePort{port}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: cr.Namespace,
		Labels:    labels,
	}, Spec: corev1.ServiceSpec{
		Ports:    ports,
		Selector: labels,
		Type:     corev1.ServiceTypeClusterIP},
	}
	return svc
}

func newJenkinsPvc(cr *jenkinsv1alpha1.Jenkins, name string) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	var quantity = resource.MustParse(JenkinsPvcSize)
	resources := corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: quantity}}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources:   resources,
		},
	}

	return pvc
}
