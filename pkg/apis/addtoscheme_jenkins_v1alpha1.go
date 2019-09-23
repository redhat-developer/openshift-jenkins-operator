package apis

import (
	authv1 "github.com/openshift/api/authorization/v1"
	"github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, authv1.AddToScheme)
}
