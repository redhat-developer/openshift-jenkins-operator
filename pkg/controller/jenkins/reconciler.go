package jenkins

import (
	"context"

	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	common "github.com/redhat-developer/openshift-jenkins-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	JenkinsInstanceName = ""
	logReconciler       = logf.Log.WithName("jenkins/reconciler.go")
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
	JenkinsPvcDefaultSize  = "1Gi"
	JenkinsVolumeName      = "jenkins-data"
	JenkinsVolumeMountPath = "/var/lib/jenkins"
)

// ReconcileJenkins reconciles a Jenkins object
type JenkinsReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client               client.Client
	Scheme               *runtime.Scheme
	Result               reconcile.Result
	Request              reconcile.Request
	ControlledRescources ControlledResources
	Messages             common.Messages
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, messages common.Messages) reconcile.Reconciler {
	return &JenkinsReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Messages: messages, ControlledRescources: ControlledResources{}}
}

/*
	Reconciliation && Requeing Requests : Works on matching the current state of the resources to the expected state
	The Controller will requeue the Request to be processed again if the returned error is non-nil
*/
func (r *JenkinsReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Record Request and Jenkins Instance Name
	r.Request = request
	JenkinsInstanceName = request.NamespacedName.Name
	// Get the Jenkins Instance
	r.ControlledRescources.JenkinsInstance = &jenkinsv1alpha1.Jenkins{}
	err := r.Client.Get(context.TODO(), r.Request.NamespacedName, r.ControlledRescources.JenkinsInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, nil
	}
	// Create Resources
	r.createAllResources()

	// Resources on Watch
	resourcesToWatch := []NamedResource{
		NamedResource{r.ControlledRescources.ServiceAccount, r.ControlledRescources.ServiceAccount.GetName()},
		NamedResource{r.ControlledRescources.RoleBinding, r.ControlledRescources.RoleBinding.GetName()},
		NamedResource{r.ControlledRescources.JenkinsService, r.ControlledRescources.JenkinsService.GetName()},
		NamedResource{r.ControlledRescources.JNLPService, r.ControlledRescources.JNLPService.GetName()},
		NamedResource{r.ControlledRescources.Route, r.ControlledRescources.Route.GetName()},
		//NamedResource{r.ControlledRescources.DeploymentConfig, r.ControlledRescources.DeploymentConfig.GetName()},
		NamedResource{r.ControlledRescources.Deployment, r.ControlledRescources.Deployment.GetName()},
	}

	if r.isPersistent() {
		r.ControlledRescources.PersistentVolumeClaim = newJenkinsPvc(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName)
		resourcesToWatch = append(resourcesToWatch, NamedResource{r.ControlledRescources.PersistentVolumeClaim, r.ControlledRescources.PersistentVolumeClaim.GetName()})
	}

	// Set reference and watch resources
	r.setControllerReferenceOnWatch(resourcesToWatch)
	r.updateResourcesOnWatch(resourcesToWatch)

	return r.Result, err
}

func (r *JenkinsReconciler) setControllerReferenceOnWatch(resourcesToWatch []NamedResource) {
	// Set Controller reference as Jenkins Instance
	for _, namedRes := range resourcesToWatch {
		resource := r.parseResourceToStatic(namedRes, r.ControlledRescources.JenkinsInstance.GetNamespace())
		if err := controllerutil.SetControllerReference(r.ControlledRescources.JenkinsInstance, resource.Object, r.Scheme); err != nil {
			r.Messages.LogError(err, "setControllerReferenceOnWatch", logReconciler)
		} else {
			r.Result = r.Messages.HandleComplete()
		}
	}
}

func (r *JenkinsReconciler) updateResourcesOnWatch(resourcesToWatch []NamedResource) {
	// Watch resources and create if they aren't present
	for _, namedRes := range resourcesToWatch {
		resource := r.parseResourceToRuntime(namedRes, r.ControlledRescources.JenkinsInstance.GetNamespace())
		if err := r.createResourceIfNotPresent(resource); err != nil {
			r.Messages.LogError(err, "updateResourcesOnWatch", logReconciler)
		} else {
			r.Result = r.Messages.HandleComplete()
		}
	}
}

func (r *JenkinsReconciler) createAllResources() {
	// Define a DeploymentConfig
	//r.ControlledRescources.DeploymentConfig = newJenkinsDeploymentConfig(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName, JenkinsInstanceName+JenkinsJnlpServiceSuffix, r.ControlledRescources.JenkinsInstance.Spec.Persistence.Enabled)

	// Define a Deployment
	r.ControlledRescources.Deployment = newJenkinsDeployment(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName, JenkinsInstanceName+JenkinsJnlpServiceSuffix, r.ControlledRescources.JenkinsInstance.Spec.Persistence.Enabled)
	// Define Jenkins Services
	r.ControlledRescources.JenkinsService = r.getJenkinsService()
	r.ControlledRescources.JNLPService = r.getJenkinsJNLPService()
	// Define Route
	r.ControlledRescources.Route = newJenkinsRoute(r.ControlledRescources.JenkinsInstance, r.ControlledRescources.JenkinsService)

	// Create RBAC and manage
	r.ControlledRescources.ServiceAccount = newJenkinsServiceAccount(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName)
	r.ControlledRescources.RoleBinding = newJenkinsRoleBinding(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName)
}

func (r *JenkinsReconciler) getJenkinsService() *corev1.Service {
	jenkinsPort := corev1.ServicePort{
		Name:     JenkinsWebPortName,
		Port:     JenkinsWebPort,
		Protocol: JenkinsWebPortProtocol,
		TargetPort: intstr.IntOrString{
			IntVal: JenkinsWebPortAsInt,
			StrVal: JenkinsWebPortAsStr,
		},
	}
	return newJenkinsService(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName, jenkinsPort)
}

func (r *JenkinsReconciler) getJenkinsJNLPService() *corev1.Service {
	jenkinsJNLPPort := corev1.ServicePort{
		Name:     JenkinsAgentPortName,
		Port:     JenkinsAgentPort,
		Protocol: JenkinsAgentPortProtocol,
		TargetPort: intstr.IntOrString{
			IntVal: JenkinsAgentPort,
			StrVal: JenkinsAgentPortAsStr,
		},
	}
	return newJenkinsService(r.ControlledRescources.JenkinsInstance, JenkinsInstanceName+JenkinsJnlpServiceSuffix, jenkinsJNLPPort)
}

func (r *JenkinsReconciler) createResourceIfNotPresent(resource RuntimeResource) error {
	namespaceNameLog := "| Namespace " + resource.NamespacedName.Namespace + " | Name " + resource.NamespacedName.Name
	message := "createResourceIfNotPresent: " + namespaceNameLog
	noRequeueMessage := message + " REQUEUE DISABLED "

	r.Messages.LogInfo(message, logReconciler)
	err := r.checkResourceIfExists(resource)
	if err != nil && kubeerrors.IsNotFound(err) {
		r.Messages.LogError(err, message, logReconciler)
		r.createResource(resource)
		// Resource is Successfully Created | DO NOT REQUEUE
		r.Messages.LogInfo(noRequeueMessage, logReconciler)
		r.Result = reconcile.Result{Requeue: false}
		return err
	}
	return err
}

func (r *JenkinsReconciler) checkResourceIfExists(resource RuntimeResource) error {
	namespaceNameLog := "| Namespace " + resource.NamespacedName.Namespace + " | Name " + resource.NamespacedName.Name
	message := "checkResourceIfExists: " + namespaceNameLog

	r.Messages.LogInfo(message, logReconciler)
	err := r.Client.Get(context.TODO(), resource.NamespacedName, resource.Object)
	if err != nil && kubeerrors.IsAlreadyExists(err) {
		r.Messages.LogError(err, message, logReconciler)
		return err
	}
	return err
}

func (r *JenkinsReconciler) isPersistent() bool {
	return r.ControlledRescources.JenkinsInstance.Spec.Persistence.Enabled
}

func (r *JenkinsReconciler) createResource(resource RuntimeResource) {
	namespaceNameLog := "| Namespace " + resource.NamespacedName.Namespace + " | Name " + resource.NamespacedName.Name
	message := "createResource: " + namespaceNameLog
	requeueMessage := message + " REQUEUE ENABLED "

	r.Messages.LogInfo(message, logReconciler)
	err := r.Client.Create(context.TODO(), resource.Object)
	if err != nil {
		r.Messages.LogError(err, message, logReconciler)
		r.Messages.LogInfo(requeueMessage, logReconciler)
		r.Result = reconcile.Result{Requeue: true}
	}
}

func (r *JenkinsReconciler) parseResourceToStatic(namedRes NamedResource, namespace string) StaticResource {
	resource := namedRes.Object.(metav1.Object)
	return StaticResource{
		Object: resource,
		NamespacedName: types.NamespacedName{
			Name:      namedRes.Name,
			Namespace: namespace,
		},
	}
}

func (r *JenkinsReconciler) parseResourceToRuntime(namedRes NamedResource, namespace string) RuntimeResource {
	resource := namedRes.Object.(runtime.Object)
	return RuntimeResource{
		Object: resource,
		NamespacedName: types.NamespacedName{
			Name:      namedRes.Name,
			Namespace: namespace,
		},
	}
}
