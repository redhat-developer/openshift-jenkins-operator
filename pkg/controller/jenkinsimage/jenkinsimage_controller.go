package jenkinsimage

import (
	"context"

	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	//appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	buildv1 "github.com/openshift/api/build/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)


const registryHostname = "image-registry.openshift-image-registry.svc:5000"

var log = logf.Log.WithName("controller_jenkinsimage")

// Add creates a new JenkinsImage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileJenkinsImage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkinsimage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource JenkinsImage
	err = c.Watch(&source.Kind{Type: &jenkinsv1alpha1.JenkinsImage{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource and requeue the owner JenkinsImage
	err = c.Watch(&source.Kind{Type: &imagev1.ImageStream{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.JenkinsImage{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource and requeue the owner JenkinsImage
	err = c.Watch(&source.Kind{Type: &buildv1.BuildConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &jenkinsv1alpha1.JenkinsImage{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileJenkinsImage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkinsImage{}

// ReconcileJenkinsImage reconciles a JenkinsImage object
type ReconcileJenkinsImage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a JenkinsImage object and makes changes based on the state read
// and what is in the JenkinsImage.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkinsImage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsImage")

	// Fetch the JenkinsImage instance
	instance := &jenkinsv1alpha1.JenkinsImage{}
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

	// Define an image stream object
	imagestream := newImageStream(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, imagestream, r.scheme); err != nil {
	return reconcile.Result{}, err
	}
	// Check if this ImageStream already exists
	isFound := &imagev1.ImageStream{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: imagestream.Name, Namespace: imagestream.Namespace}, isFound)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ImageStream", "ImageStream.Namespace", imagestream.Namespace, "ImageStream.Name", imagestream.Name)
		err = r.client.Create(context.TODO(), imagestream)
		if err != nil {
			return reconcile.Result{}, err
		}
		// ImageStream created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}


	// Define a new buildConfig object
	buildConfig := newBuildConfig(instance)
	// Set JenkinsImage instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, buildConfig, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this BuildConfig already exists
	found := &buildv1.BuildConfig{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: buildConfig.Name, Namespace: buildConfig.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new BuildConfig", "BuildConfig.Namespace", buildConfig.Namespace, "BuildConfig.Name", buildConfig.Name)
		err = r.client.Create(context.TODO(), buildConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
		// BuildConfig created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// BuildConfig already exists - don't requeue
	reqLogger.Info("Skip reconcile: BuildConfig already exists", "BuildConfig.Namespace", found.Namespace, "BuildConfig.Name", found.Name)
	return reconcile.Result{}, nil
}



// newBuildConfg returns a BuildConfig with binatry source strategy using the source image or imagestream specified in the CR
func newImageStream(cr *jenkinsv1alpha1.JenkinsImage) *imagev1.ImageStream {
	// create imagerepo
	imageName := registryHostname + "/" + cr.Namespace + "/" + cr.Name
	is := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Name, 
			Namespace: cr.Namespace,
		},
		Spec: imagev1.ImageStreamSpec{
				DockerImageRepository: imageName,
				Tags: []imagev1.TagReference{
						{
								Name: "latest",
								From: &corev1.ObjectReference{
										Kind: "DockerImage",
										Name: imageName,
								},
						},
				},
		},
	}
	return is
}

// newBuildConfg returns a BuildConfig with binatry source strategy using the source image or imagestream specified in the CR
func newBuildConfig(cr *jenkinsv1alpha1.JenkinsImage) *buildv1.BuildConfig {
	bc := &buildv1.BuildConfig{		
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name,
				Namespace: cr.Namespace,
			},
			Spec: buildv1.BuildConfigSpec{
				RunPolicy: buildv1.BuildRunPolicySerial,
				CommonSpec: buildv1.CommonSpec{
					Source: buildv1.BuildSource{
						Binary: &buildv1.BinaryBuildSource{},
					},
					Strategy: buildv1.BuildStrategy{
						SourceStrategy: &buildv1.SourceBuildStrategy{
							From: corev1.ObjectReference{
									Kind: "ImageStreamTag",
									Name: "jenkins" + ":" + "2",
									Namespace: "openshift",
							},
						},
					},
					Output: buildv1.BuildOutput{
						To: &corev1.ObjectReference{
								Kind: "ImageStreamTag",
								Name: cr.Name + ":latest",
						},
					},
				},
			},
		}
	return bc
}
