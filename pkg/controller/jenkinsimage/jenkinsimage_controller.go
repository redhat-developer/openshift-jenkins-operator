package jenkinsimage

import (
	"bytes"
	"context"
	"fmt"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	cu "github.com/redhat-developer/openshift-jenkins-operator/pkg/controller/controllerutil"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	ImageStreamTagKind      = "ImageStreamTag"
	DockerImageKind         = "DockerImage"
	ImageToTagSeparator     = ":"
    ImageNameSeparator      = "/"
    DefaultRegistryHostname = "image-registry.openshift-image-registry.svc:5000"
	DefaultImageNamespace   = "openshift"
	DefaultJenkinsBaseImage = "jenkins" + ":" + "2"
	DefaultImageStreamTag   = "latest"
	PluginsListFilename     = "plugins.txt"

	OcCommand = "oc"
	StartBuildArg = "start-build"
	FromDirArg = "--from-dir"
)

var log = logf.Log.WithName("jenkinsimage_controller")

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
	// Create owner reference stating the owner of all the resources under the controller
	ownerRef := &jenkinsv1alpha1.JenkinsImage{}
	resourcesToWatch := []cu.NamedResource{
		cu.NamedResource{ownerRef, ""},
		cu.NamedResource{&imagev1.ImageStream{}, ""},
		cu.NamedResource{&buildv1.BuildConfig{}, ""},
	}
	for _, resource := range resourcesToWatch {
		ownerReference := ownerRef
		if reflect.DeepEqual(resource.Object, ownerRef) {
			ownerReference = nil
		}
		cu.WatchResourceOrStackError(c, resource, ownerReference)
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

// The Controller will requeue the request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkinsImage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling JenkinsImage")

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
		logger.Info("Creating a new ImageStream", "ImageStream.Namespace", imagestream.Namespace, "ImageStream.Name", imagestream.Name)
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
		logger.Info("Creating a new BuildConfig", "BuildConfig.Namespace", buildConfig.Namespace, "BuildConfig.Name", buildConfig.Name)
		err = r.client.Create(context.TODO(), buildConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
		// BuildConfig created successfully - don't requeue and start the binary build from temp dir
		startBinaryBuild(instance, buildConfig)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// BuildConfig already exists - don't requeue
	logger.Info("Skip reconcile: BuildConfig already exists", "BuildConfig.Namespace", found.Namespace, "BuildConfig.Name", found.Name)
	return reconcile.Result{}, nil
}

func startBinaryBuild(instance *jenkinsv1alpha1.JenkinsImage, bc *buildv1.BuildConfig) {
	name := bc.Name
	logger := log.WithName("jenkinsimage_startbinarybuild")
	tmpDir, err := ioutil.TempDir("","prefix")
	if err != nil {
		logger.Error(err, "Error")
		fmt.Println(err)
	}
	defer os.RemoveAll(tmpDir)

	pluginsFile, err := os.Create(filepath.Join(tmpDir, filepath.Base(PluginsListFilename)))
	if err != nil {
		logger.Error(err, fmt.Sprint("Error while creating tmp file for binary build: ", pluginsFile , err))
		fmt.Println(err)
		pluginsFile.Close()
		return
	}
	defer pluginsFile.Close()
	for _, v := range instance.Spec.Plugins {
		fmt.Fprintf(pluginsFile, "%s:%s", v.Name, v.Version)
		logger.Info(fmt.Sprintf("Writing value %s into file %s ", pluginsFile.Name() , v.Name))
		if err != nil {
			logger.Error(err, fmt.Sprint("Error while writing plugins list into tmpFile: ", pluginsFile , err))
			return
		}
	}
	err = pluginsFile.Close()
	if err != nil {
		logger.Error(err, fmt.Sprint("Error while closing plugins file: ", pluginsFile , err))
		return
	}

	cmd := exec.Command(OcCommand, StartBuildArg, name, FromDirArg, tmpDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		logger.Error(err, fmt.Sprint("oc start-build command failed with error: ", cmd))
	}
}
