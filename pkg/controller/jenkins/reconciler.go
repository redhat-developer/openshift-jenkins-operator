package jenkins

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "github.com/openshift/api/apps/v1"
	authv1 "github.com/openshift/api/authorization/v1"
	routev1 "github.com/openshift/api/route/v1"
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
	JenkinsContainerMemory   = "1Gi"
	JenkinsAppLabel          = "app"
	JenkinsNameLabel         = "name"

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
		return Done()
	}

	// Define a new DC object
	jenkinsDc := newJenkinsDeploymentConfig(instance, jenkinsInstanceName, jenkinsInstanceName+JenkinsJnlpServiceSuffix)

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
	jenkinsRoute := newJenkinsRoute(instance, jenkinsSvc)
	jenkinsPvc := newJenkinsPvc(instance, jenkinsInstanceName) // jenknis pvc
	jenkinsServiceAccount := newJenkinsServiceAccount(instance, jenkinsInstanceName)
	jenkinsRoleBinding := newJenkinsRoleBinding(instance, jenkinsInstanceName)

	// Set Jenkins instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, jenkinsDc, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsSvc, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsJNLPSvc, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsRoute, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsPvc, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsServiceAccount, r.scheme); err != nil {
		return Done()
	}
	if err := controllerutil.SetControllerReference(instance, jenkinsRoleBinding, r.scheme); err != nil {
		return Done()
	}

	// TODO implement error checking for existence of the resource
	// *or if there is an actual error* deal with it (<>_<>-)
	// Also -- https://github.com/redhat-developer/openshift-jenkins-operator/pull/17#pullrequestreview-289463590 --

	err = r.createResourceIfNotPresent(jenkinsDc, jenkinsDc.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsServiceAccount, jenkinsServiceAccount.ObjectMeta.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsRoleBinding, jenkinsRoleBinding.ObjectMeta.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsSvc, jenkinsSvc.ObjectMeta.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsJNLPSvc, jenkinsJNLPSvc.ObjectMeta.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsRoute, jenkinsRoute.ObjectMeta.Name, instance.Namespace)
	err = r.createResourceIfNotPresent(jenkinsPvc, jenkinsPvc.ObjectMeta.Name, instance.Namespace)
	return r.result, nil
}

/*
Jenkins Resources created by jenkins-persistent Template

Route
Service (jnlp)
Service (jenkins)
PersistentVolumeClaim
DeploymentConfig
ServiceAccount
RoleBinding


*/

func (r *JenkinsReconciler) createResourceIfNotPresent(resource runtime.Object, name string, namespace string) error {
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	//	r.logger.Info("Checking if object exists", "in Namespace", key.Namespace, "Resource.Name", resource)
	err := r.client.Get(context.TODO(), key, resource)
	if err != nil && errors.IsAlreadyExists(err) {
		r.logger.Info("Object already exists", "in Namespace", key.Namespace, "Resource.Name", resource, ": No need to requeue")
		r.result = reconcile.Result{Requeue: false}
		return err
	}
	if err != nil && errors.IsNotFound(err) {
		r.logger.Info("Creating a new Object", "in Namespace", key.Namespace, "Resource.Name", resource)
		err = r.client.Create(context.TODO(), resource)
		if err != nil {
			r.logger.Info("Error while creating an object", "Object.Namespace", key.Namespace, "Object.Name", resource, "Error:", err)
			return err
		}
		// Resource created successfully - don't requeue
		r.result = reconcile.Result{Requeue: false}
		return nil
	}

	return nil
}

// newDeploymentConfigForCR returns a jenkins DeploymentConfig with the same name/namespace as the cr
func newJenkinsDeploymentConfig(cr *jenkinsv1alpha1.Jenkins, jenkinsService, jenkinsJNLPService string) *appsv1.DeploymentConfig {
	labels := map[string]string{
		JenkinsAppLabelName: cr.Name,
		JenkinsNameLabel:    JenkinsContainerName,
	}
	jenkinsInstanceName := cr.Name

	envVars := []corev1.EnvVar{
		corev1.EnvVar{Name: "OPENSHIFT_ENABLE_OAUTH", Value: "true"},
		corev1.EnvVar{Name: "OPENSHIFT_ENABLE_REDIRECT_PROMPT", Value: "true"},
		corev1.EnvVar{Name: "DISABLE_ADMINISTRATIVE_MONITORS", Value: "false"},
		corev1.EnvVar{Name: "KUBERNETES_MASTER", Value: "https://kubernetes.default:443"},
		corev1.EnvVar{Name: "KUBERNETES_TRUST_CERTIFICATES", Value: "true"},
		corev1.EnvVar{Name: "JENKINS_SERVICE_NAME", Value: jenkinsService},
		corev1.EnvVar{Name: "JNLP_SERVICE_NAME", Value: jenkinsJNLPService},
		corev1.EnvVar{Name: "ENABLE_FATAL_ERROR_LOG_FILE", Value: "false"},
		corev1.EnvVar{Name: "JENKINS_UC_INSECURE", Value: "false"},
	}

	livenessProbe := newProbe("/login", 8080, 420, 240, 360)
	readinessProbe := newProbe("/login", 8080, 3, 240, 0)

	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jenkinsInstanceName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labels,
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.DeploymentStrategyTypeRecreate},
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
							Env:                    envVars,
							LivenessProbe:          &livenessProbe,
							ReadinessProbe:         &readinessProbe,
							TerminationMessagePath: "/dev/termination-log",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(JenkinsContainerMemory),
								},
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
					ServiceAccountName: jenkinsInstanceName,
				},
			},
		},
	}

	return dc
}

func newProbe(path string, port int, initialDelaySeconds, timeoutSeconds, periodSeconds int32) corev1.Probe {
	probe := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt(port),
			},
		},
		FailureThreshold:    2,
		InitialDelaySeconds: initialDelaySeconds,
		TimeoutSeconds:      timeoutSeconds,
	}

	if periodSeconds > 0 {
		probe.PeriodSeconds = periodSeconds
	}

	return probe
}

// newJenkinsServicefor templates a new Service for Jenkins
func newJenkinsService(cr *jenkinsv1alpha1.Jenkins, name string, port corev1.ServicePort) *corev1.Service {
	labels := map[string]string{
		JenkinsAppLabel: cr.Name,
	}
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

func newJenkinsRoute(cr *jenkinsv1alpha1.Jenkins, svc *corev1.Service) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: routev1.RouteSpec{
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind: svc.Kind,
				Name: svc.Name,
			},
		},
	}
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

func newJenkinsServiceAccount(cr *jenkinsv1alpha1.Jenkins, name string) *corev1.ServiceAccount {
	labels := map[string]string{
		JenkinsAppLabel:  cr.Name,
		JenkinsNameLabel: JenkinsServiceName,
	}
	annotationKey := "serviceaccounts.openshift.io/oauth-redirectreference." + cr.Name
	annotationValue := "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"" + cr.Name + "\"}}"
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				annotationKey: annotationValue,
			},
		},
	}
}

func newJenkinsRoleBinding(cr *jenkinsv1alpha1.Jenkins, jenkinsServiceAccountName string) *authv1.RoleBinding {
	labels := map[string]string{
		JenkinsAppLabel:  cr.Name,
		JenkinsNameLabel: JenkinsServiceName,
	}
	return &authv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "_edit",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		RoleRef: corev1.ObjectReference{
			Name: "edit",
		},
		Subjects: []corev1.ObjectReference{
			corev1.ObjectReference{
				Kind: "ServiceAccount",
				Name: jenkinsServiceAccountName,
			},
		},
	}
}
