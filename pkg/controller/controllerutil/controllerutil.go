package controllerutil

import (
	"fmt"

	"github.com/redhat-developer/openshift-jenkins-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
	AddToManagerFuncs []func(manager.Manager) error
	logController      = logf.Log.WithName("jenkins/controller_util.go")
	controllerMessages = common.NewMessages("Jenkins Controller")
)

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


// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

// WatchResourceOrStackError watch the resource passed as resource and set owner as the parent
func WatchResourceOrStackError(controller controller.Controller, resource NamedResource, owner runtime.Object) {
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
