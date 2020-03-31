package apis

import (
	"github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
